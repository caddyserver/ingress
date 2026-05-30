package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/certmagic"
	"github.com/caddyserver/ingress/internal/k8s"
	"github.com/caddyserver/ingress/pkg/store"
	"go.uber.org/zap"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	// load required caddy plugins
	_ "github.com/caddyserver/caddy/v2/modules/caddyhttp/proxyprotocol"
	_ "github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
	_ "github.com/caddyserver/caddy/v2/modules/caddytls"
	_ "github.com/caddyserver/caddy/v2/modules/caddytls/standardstek"
	_ "github.com/caddyserver/caddy/v2/modules/metrics"
	_ "github.com/caddyserver/ingress/pkg/storage"
)

const (
	// how often we should attempt to keep ingress resource's source address in sync
	syncInterval = time.Second * 30

	// how often we resync informers resources (besides receiving updates)
	resourcesSyncInterval = time.Hour * 1
)

// Action is an interface for ingress actions.
type Action interface {
	handle(c *CaddyController) error
}

// Informer defines the required SharedIndexInformers that interact with the API server.
type Informer struct {
	ConfigMap     cache.SharedIndexInformer
	Ingress       cache.SharedIndexInformer
	Service       cache.SharedIndexInformer
	EndpointSlice cache.SharedIndexInformer
	Secret        cache.SharedIndexInformer
}

// InformerFactory contains shared informer factory
// We need to type of factory:
// - One used to watch ConfigMap and Secret resources
// - Another one for Ingress resources in the selected namespace
type InformerFactory struct {
	ConfigNamespace  informers.SharedInformerFactory
	WatchedNamespace informers.SharedInformerFactory
}

type Converter interface {
	ConvertToCaddyConfig(store *store.Store) (any, error)
}

// CaddyController represents a caddy ingress controller.
type CaddyController struct {
	resourceStore *store.Store

	kubeClient *kubernetes.Clientset

	logger *zap.SugaredLogger

	// main queue syncing ingresses, configmaps, ... with caddy
	syncQueue workqueue.TypedRateLimitingInterface[Action]

	// informer factories
	factories *InformerFactory

	// informer contains the cache Informers
	informers *Informer

	// save last applied caddy config
	lastAppliedConfig []byte

	converter Converter

	stopChan chan struct{}
}

func NewCaddyController(
	logger *zap.SugaredLogger,
	kubeClient *kubernetes.Clientset,
	opts store.Options,
	converter Converter,
	stopChan chan struct{},
) *CaddyController {
	controller := &CaddyController{
		logger:     logger,
		kubeClient: kubeClient,
		converter:  converter,
		stopChan:   stopChan,
		syncQueue:  workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[Action]()),
		informers:  &Informer{},
		factories:  &InformerFactory{},
	}

	podInfo, err := k8s.GetPodDetails(logger, kubeClient)
	if err != nil {
		logger.Fatalf("Unexpected error obtaining pod information: %v", err)
	}

	var configNamespace, configMapName string
	if parts := strings.SplitN(opts.ConfigMapName, "/", 2); len(parts) == 2 {
		configNamespace, configMapName = parts[0], parts[1]
	} else if podInfo != nil {
		configNamespace, configMapName = podInfo.Namespace, opts.ConfigMapName
	} else {
		logger.Fatalf("Must set a namespace for -config-map when running outside a cluster: %s", opts.ConfigMapName)
	}
	opts.ConfigMapName = configMapName

	// Create informer factories
	controller.factories.ConfigNamespace = informers.NewSharedInformerFactoryWithOptions(
		kubeClient,
		resourcesSyncInterval,
		informers.WithNamespace(configNamespace),
	)
	controller.factories.WatchedNamespace = informers.NewSharedInformerFactoryWithOptions(
		kubeClient,
		resourcesSyncInterval,
		informers.WithNamespace(opts.WatchNamespace),
	)

	// Create informers
	controller.watchConfigMap()
	controller.watchIngresses()
	controller.watchServices()
	controller.watchEndpointSlices()
	controller.watchSecrets()

	// Create resource store
	controller.resourceStore, err = store.NewStore(
		logger,
		kubeClient,
		opts,
		configNamespace,
		podInfo,
		controller.informers.Ingress.GetIndexer(),
		controller.informers.Service.GetIndexer(),
		controller.informers.EndpointSlice.GetIndexer(),
		controller.informers.Secret.GetIndexer(),
	)
	if err != nil {
		logger.Fatalf("Unexpected error initializing store: %v", err)
	}

	return controller
}

// Shutdown stops the caddy controller.
func (c *CaddyController) Shutdown() error {
	// remove this ingress controller's ip from ingress resources.
	c.updateIngStatuses([]networkingv1.IngressLoadBalancerIngress{{}}, c.resourceStore.Ingresses())

	if err := caddy.Stop(); err != nil {
		c.logger.Error("failed to stop caddy server", zap.Error(err))
		return err
	}
	certmagic.CleanUpOwnLocks(context.TODO(), c.logger.Desugar())
	return nil
}

// Run method starts the ingress controller.
func (c *CaddyController) Run() {
	defer runtime.HandleCrash()
	defer c.syncQueue.ShutDown()

	// start informers where we listen to new / updated resources
	c.factories.ConfigNamespace.Start(c.stopChan)
	c.factories.WatchedNamespace.Start(c.stopChan)

	// wait for all involved caches to be synced before processing items
	// from the queue
	c.factories.ConfigNamespace.WaitForCacheSync(c.stopChan)
	c.factories.WatchedNamespace.WaitForCacheSync(c.stopChan)

	// start processing events for syncing ingress resources
	go wait.Until(c.runWorker, time.Second, c.stopChan)

	// start ingress status syncher and run every syncInterval
	go wait.Until(c.dispatchSync, syncInterval, c.stopChan)

	// wait for SIGTERM
	<-c.stopChan
	c.logger.Info("stopping ingress controller")

	var exitCode int
	err := c.Shutdown()
	if err != nil {
		c.logger.Error("could not shutdown ingress controller properly, " + err.Error())
		exitCode = 1
	}

	os.Exit(exitCode)
}

// runWorker processes items in the event queue.
func (c *CaddyController) runWorker() {
	for c.processNextItem() {
	}
}

// processNextItem determines if there is an ingress item in the event queue and processes it.
func (c *CaddyController) processNextItem() bool {
	// Wait until there is a new item in the working queue
	action, quit := c.syncQueue.Get()
	if quit {
		return false
	}

	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two ingresses with the same key are never processed in
	// parallel.
	defer c.syncQueue.Done(action)

	// Invoke the method containing the business logic
	err := action.handle(c)
	if err != nil {
		c.handleErr(err, action)
		return true
	}

	err = c.reloadCaddy()
	if err != nil {
		c.logger.Error("could not reload caddy: " + err.Error())
		return true
	}

	return true
}

// handleErrs reports errors received from queue actions.
//
//goland:noinspection GoUnusedParameter
func (c *CaddyController) handleErr(err error, action any) {
	c.logger.Error(err.Error())
}

// reloadCaddy generate a caddy config from controller's store
func (c *CaddyController) reloadCaddy() error {
	config, err := c.converter.ConvertToCaddyConfig(c.resourceStore)
	if err != nil {
		return err
	}

	j, err := json.Marshal(config)
	if err != nil {
		return err
	}

	if bytes.Equal(c.lastAppliedConfig, j) {
		c.logger.Debug("caddy config did not change, skipping reload")
		return nil
	}

	c.logger.Debug("reloading caddy with config", string(j))
	err = caddy.Load(j, false)
	if err != nil {
		return fmt.Errorf("could not reload caddy config %v", err.Error())
	}
	c.lastAppliedConfig = j
	return nil
}
