package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/ingress/pkg/k8s"
	"github.com/caddyserver/ingress/pkg/storage"
	"github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"os"
	"time"

	// load required caddy plugins
	_ "github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
	_ "github.com/caddyserver/caddy/v2/modules/caddytls"
	_ "github.com/caddyserver/caddy/v2/modules/caddytls/standardstek"
	_ "github.com/caddyserver/ingress/pkg/proxy"
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

// Options represents ingress controller config received through cli arguments.
type Options struct {
	WatchNamespace string
	ConfigMapName  string
}

// Store contains resources used to generate Caddy config
type Store struct {
	Options   *Options
	ConfigMap *k8s.ConfigMapOptions
	Ingresses []*v1beta1.Ingress
}

// Informer defines the required SharedIndexInformers that interact with the API server.
type Informer struct {
	Ingress   cache.SharedIndexInformer
	ConfigMap cache.SharedIndexInformer
	TLSSecret cache.SharedIndexInformer
}

// InformerFactory contains shared informer factory
// We need to type of factory:
// - One used to watch resources in the Pod namespaces (caddy config, secrets...)
// - Another one for Ingress resources in the selected namespace
type InformerFactory struct {
	PodNamespace     informers.SharedInformerFactory
	WatchedNamespace informers.SharedInformerFactory
}

type Converter interface {
	ConvertToCaddyConfig(namespace string, store *Store) (interface{}, error)
}

// CaddyController represents an caddy ingress controller.
type CaddyController struct {
	resourceStore *Store

	kubeClient *kubernetes.Clientset

	// main queue syncing ingresses, configmaps, ... with caddy
	syncQueue workqueue.RateLimitingInterface

	// informer factories
	factories *InformerFactory

	// informer contains the cache Informers
	informers *Informer

	// ingress controller pod infos
	podInfo *k8s.Info

	// save last applied caddy config
	lastAppliedConfig []byte

	converter Converter

	stopChan chan struct{}
}

func NewCaddyController(kubeClient *kubernetes.Clientset, opts Options, converter Converter) *CaddyController {
	controller := &CaddyController{
		kubeClient: kubeClient,
		converter:  converter,
		syncQueue:  workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		informers:  &Informer{},
		factories:  &InformerFactory{},
	}

	podInfo, err := k8s.GetPodDetails(kubeClient)
	if err != nil {
		logrus.Fatalf("Unexpected error obtaining pod information: %v", err)
	}
	controller.podInfo = podInfo

	// Create informer factories
	controller.factories.PodNamespace = informers.NewSharedInformerFactoryWithOptions(
		kubeClient,
		resourcesSyncInterval,
		informers.WithNamespace(controller.podInfo.Namespace),
	)
	controller.factories.WatchedNamespace = informers.NewSharedInformerFactoryWithOptions(
		kubeClient,
		resourcesSyncInterval,
		informers.WithNamespace(opts.WatchNamespace),
	)

	// Watch ingress resources in selected namespaces
	ingressParams := k8s.IngressParams{
		InformerFactory: controller.factories.WatchedNamespace,
		// TODO Add configuration for that
		ClassName:         "caddy",
		ClassNameRequired: false,
	}
	controller.informers.Ingress = k8s.WatchIngresses(ingressParams, k8s.IngressHandlers{
		AddFunc:    controller.onIngressAdded,
		UpdateFunc: controller.onIngressUpdated,
		DeleteFunc: controller.onIngressDeleted,
	})

	// Watch Configmap in the pod's namespace for global options
	cmOptionsParams := k8s.ConfigMapParams{
		Namespace:       podInfo.Namespace,
		InformerFactory: controller.factories.PodNamespace,
		ConfigMapName:   opts.ConfigMapName,
	}
	controller.informers.ConfigMap = k8s.WatchConfigMaps(cmOptionsParams, k8s.ConfigMapHandlers{
		AddFunc:    controller.onConfigMapAdded,
		UpdateFunc: controller.onConfigMapUpdated,
		DeleteFunc: controller.onConfigMapDeleted,
	})

	// register kubernetes specific cert-magic storage module and proxy module
	caddy.RegisterModule(storage.SecretStorage{})

	// Create resource store
	controller.resourceStore = NewStore(opts)

	return controller
}

// Shutdown stops the caddy controller.
func (c *CaddyController) Shutdown() error {
	// remove this ingress controller's ip from ingress resources.
	c.updateIngStatuses([]apiv1.LoadBalancerIngress{{}}, c.resourceStore.Ingresses)
	return nil
}

// Run method starts the ingress controller.
func (c *CaddyController) Run(stopCh chan struct{}) {
	defer runtime.HandleCrash()
	defer c.syncQueue.ShutDown()

	// start informers where we listen to new / updated resources
	go c.informers.ConfigMap.Run(stopCh)
	go c.informers.Ingress.Run(stopCh)

	// wait for all involved caches to be synced before processing items
	// from the queue
	if !cache.WaitForCacheSync(stopCh,
		c.informers.ConfigMap.HasSynced,
		c.informers.Ingress.HasSynced,
	) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
	}

	// start processing events for syncing ingress resources
	go wait.Until(c.runWorker, time.Second, stopCh)

	// start ingress status syncher and run every syncInterval
	go wait.Until(c.dispatchSync, syncInterval, stopCh)

	// wait for SIGTERM
	<-stopCh
	logrus.Info("stopping ingress controller")

	var exitCode int
	err := c.Shutdown()
	if err != nil {
		logrus.Errorf("could not shutdown ingress controller properly, %v", err.Error())
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
	err := action.(Action).handle(c)
	if err != nil {
		c.handleErr(err, action)
		return true
	}

	err = c.reloadCaddy()
	if err != nil {
		logrus.Errorf("could not reload caddy: %v", err.Error())
		return true
	}

	return true
}

// handleErrs reports errors received from queue actions.
func (c *CaddyController) handleErr(err error, action interface{}) {
	logrus.Error(err)
}

// reloadCaddy generate a caddy config from controller's store
func (c *CaddyController) reloadCaddy() error {
	config, err := c.converter.ConvertToCaddyConfig(c.podInfo.Namespace, c.resourceStore)
	if err != nil {
		return err
	}

	j, err := json.Marshal(config)
	if err != nil {
		return err
	}

	if bytes.Equal(c.lastAppliedConfig, j) {
		logrus.Debug("caddy config did not change, skipping reload")
		return nil
	}

	logrus.Debugf("reloading caddy with config %v", string(j))
	err = caddy.Load(j, false)
	if err != nil {
		return fmt.Errorf("could not reload caddy config %v", err.Error())
	}
	c.lastAppliedConfig = j
	return nil
}
