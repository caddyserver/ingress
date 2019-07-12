package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/caddyserver/caddy/v2"
	c "github.com/caddyserver/ingress/internal/caddy"
	"github.com/caddyserver/ingress/internal/pod"
	"github.com/caddyserver/ingress/internal/store"
	"github.com/caddyserver/ingress/pkg/storage"
	"github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	// load required caddy plugins
	_ "github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
	_ "github.com/caddyserver/caddy/v2/modules/caddytls"
	_ "github.com/caddyserver/caddy/v2/modules/caddytls/standardstek"
)

const (
	// how often we should attempt to keep ingress resource's source address in sync
	syncInterval = time.Second * 30

	// we can sync secrets every hour since we still have events listening on updated, deletes, etc
	secretSyncInterval = time.Hour * 1
)

// CaddyController represents an caddy ingress controller.
type CaddyController struct {
	resourceStore  *store.Store
	kubeClient     *kubernetes.Clientset
	restClient     rest.Interface
	indexer        cache.Indexer
	syncQueue      workqueue.RateLimitingInterface
	statusQueue    workqueue.RateLimitingInterface // statusQueue performs ingress status updates every 60 seconds but inserts the work into the sync queue
	informer       cache.Controller
	certManager    *CertManager
	podInfo        *pod.Info
	config         c.ControllerConfig
	usingConfigMap bool
	stopChan       chan struct{}
}

// NewCaddyController returns an instance of the caddy ingress controller.
func NewCaddyController(kubeClient *kubernetes.Clientset, restClient rest.Interface, cfg c.ControllerConfig) *CaddyController {
	controller := &CaddyController{
		kubeClient:  kubeClient,
		syncQueue:   workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		statusQueue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		config:      cfg,
	}

	podInfo, err := pod.GetPodDetails(kubeClient)
	if err != nil {
		logrus.Fatalf("Unexpected error obtaining pod information: %v", err)
	}
	controller.podInfo = podInfo

	// load caddy config from file if mounted with config map
	var caddyCfgMap *c.Config
	cfgPath := "/etc/caddy/config.json"
	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		controller.usingConfigMap = true

		file, err := os.Open(cfgPath)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		b, err := ioutil.ReadAll(file)
		if err != nil {
			log.Fatal(err)
		}

		// load config file into caddy
		controller.syncQueue.Add(LoadConfigAction{config: bytes.NewReader(b)})
		json.Unmarshal(b, &caddyCfgMap)
	}

	// setup the ingress controller and start watching resources
	ingressListWatcher := cache.NewListWatchFromClient(restClient, "ingresses", cfg.WatchNamespace, fields.Everything())
	indexer, informer := cache.NewIndexerInformer(ingressListWatcher, &v1beta1.Ingress{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.onResourceAdded,
		UpdateFunc: controller.onResourceUpdated,
		DeleteFunc: controller.onResourceDeleted,
	}, cache.Indexers{})
	controller.indexer = indexer
	controller.informer = informer

	// setup store to keep track of resources
	controller.resourceStore = store.NewStore(controller.kubeClient, podInfo.Namespace, cfg, caddyCfgMap)

	// attempt to do initial sync of status addresses with ingresses
	controller.dispatchSync()

	// register kubernetes specific cert-magic storage module
	caddy.RegisterModule(caddy.Module{
		Name: "caddy.storage.secret_store",
		New: func() interface{} {
			return &storage.SecretStorage{
				Namespace:  podInfo.Namespace,
				KubeClient: kubeClient,
			}
		},
	})

	return controller
}

// Shutdown stops the caddy controller.
func (c *CaddyController) Shutdown() error {
	// remove this ingress controller's ip from ingress resources.
	c.updateIngStatuses([]apiv1.LoadBalancerIngress{apiv1.LoadBalancerIngress{}}, c.resourceStore.Ingresses)
	return nil
}

// Run method starts the ingress controller.
func (c *CaddyController) Run(stopCh chan struct{}) {
	err := c.reloadCaddy()
	if err != nil {
		logrus.Errorf("initial caddy config load failed, %v", err.Error())
	}

	defer runtime.HandleCrash()
	defer c.syncQueue.ShutDown()
	defer c.statusQueue.ShutDown()

	// start the ingress informer where we listen to new / updated ingress resources
	go c.informer.Run(stopCh)

	// wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	// start processing events for syncing ingress resources
	go wait.Until(c.runWorker, time.Second, stopCh)

	// start ingress status syncher and run every syncInterval
	go wait.Until(c.dispatchSync, syncInterval, stopCh)

	// wait for SIGTERM
	<-stopCh
	logrus.Info("stopping ingress controller")

	var exitCode int
	err = c.Shutdown()
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
	}

	return true
}

// handleErrs reports errors received from queue actions.
func (c *CaddyController) handleErr(err error, action interface{}) {
	logrus.Error(err)
}

// loadConfigFromFile loads caddy with a config defined by an io.Reader.
func (c *CaddyController) loadConfigFromFile(cfg io.Reader) error {
	err := caddy.Load(cfg)
	if err != nil {
		return fmt.Errorf("could not load caddy config %v", err.Error())
	}

	return nil
}

// reloadCaddy reloads the internal caddy instance with config from the internal store.
func (c *CaddyController) reloadCaddy() error {
	j, err := json.Marshal(c.resourceStore.CaddyConfig)
	if err != nil {
		return err
	}

	// DEBUG ONLY
	// PRETTY PRINT CADDY CONFIG ON UPDATE
	js, _ := json.MarshalIndent(c.resourceStore.CaddyConfig, "", "\t")
	fmt.Println(string(js))
	//

	r := bytes.NewReader(j)
	err = caddy.Load(r)
	if err != nil {
		return fmt.Errorf("could not reload caddy config %v", err.Error())
	}

	return nil
}
