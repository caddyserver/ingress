package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/caddyserver/caddy/v2"
	c "github.com/caddyserver/ingress/internal/caddy"
	"github.com/caddyserver/ingress/internal/pod"
	"github.com/caddyserver/ingress/internal/store"
	"github.com/caddyserver/ingress/pkg/storage"
	"github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
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

// Informer defines the required SharedIndexInformers that interact with the API server.
type Informer struct {
	Ingress   cache.SharedIndexInformer
	ConfigMap cache.SharedIndexInformer
}

// Lister contains object listers (stores).
type Listers struct {
	Ingress   cache.Store
	ConfigMap cache.Store
}

// CaddyController represents an caddy ingress controller.
type CaddyController struct {
	resourceStore *store.Store

	kubeClient *kubernetes.Clientset

	// main queue syncing ingresses, configmaps, ... with caddy
	syncQueue workqueue.RateLimitingInterface

	// informer contains the cache Informers
	informers *Informer

	// listers contains the cache.Store interfaces used in the ingress controller
	listers *Listers

	// cert manager manage user provided certs
	certManager *CertManager

	// ingress controller pod infos
	podInfo *pod.Info

	// config of the controller (flags)
	config c.ControllerConfig

	// if a /etc/caddy/config.json is detected, it will be used instead of ingresses
	usingConfigMap bool

	stopChan chan struct{}
}

// NewCaddyController returns an instance of the caddy ingress controller.
func NewCaddyController(kubeClient *kubernetes.Clientset, cfg c.ControllerConfig) *CaddyController {
	controller := &CaddyController{
		kubeClient: kubeClient,
		syncQueue:  workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		config:     cfg,
		informers:  &Informer{},
		listers:    &Listers{},
	}

	podInfo, err := pod.GetPodDetails(kubeClient)
	if err != nil {
		logrus.Fatalf("Unexpected error obtaining pod information: %v", err)
	}
	controller.podInfo = podInfo

	// load caddy config from file if mounted with config map
	caddyCfgMap, err := loadCaddyConfigFile("/etc/caddy/config.json")
	if err != nil {
		logrus.Fatalf("Unexpected error reading config.json: %v", err)
	}

	if caddyCfgMap != nil {
		controller.usingConfigMap = true
	}

	// create 2 types of informers: one for the caddy NS and another one for ingress resources
	ingressInformerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, syncInterval, informers.WithNamespace(cfg.WatchNamespace))
	caddyInformerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, syncInterval, informers.WithNamespace(podInfo.Namespace))

	controller.informers.Ingress = ingressInformerFactory.Networking().V1beta1().Ingresses().Informer()
	controller.listers.Ingress = controller.informers.Ingress.GetStore()

	controller.informers.ConfigMap = caddyInformerFactory.Core().V1().ConfigMaps().Informer()
	controller.listers.ConfigMap = controller.informers.ConfigMap.GetStore()

	// add event handlers
	controller.informers.Ingress.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.onResourceAdded,
		UpdateFunc: controller.onResourceUpdated,
		DeleteFunc: controller.onResourceDeleted,
	})

	controller.informers.ConfigMap.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.onConfigMapAdded,
		UpdateFunc: controller.onConfigMapUpdated,
		DeleteFunc: controller.onConfigMapDeleted,
	})

	// setup store to keep track of resources
	controller.resourceStore = store.NewStore(kubeClient, podInfo.Namespace, cfg, caddyCfgMap)

	// attempt to do initial sync of status addresses with ingresses
	controller.dispatchSync()

	// register kubernetes specific cert-magic storage module
	caddy.RegisterModule(storage.SecretStorage{})

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
	err := regenerateConfig(c)
	if err != nil {
		logrus.Errorf("initial caddy config load failed, %v", err.Error())
	}

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

func loadCaddyConfigFile(cfgPath string) (*c.Config, error) {
	var caddyCfgMap *c.Config
	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		file, err := os.Open(cfgPath)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		b, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}

		json.Unmarshal(b, &caddyCfgMap)
	} else {
		return nil, nil
	}
	return caddyCfgMap, nil
}

// reloadCaddy reloads the internal caddy instance with config from the internal store.
func (c *CaddyController) reloadCaddy(config *c.Config) error {
	j, err := json.Marshal(config)
	if err != nil {
		return err
	}

	err = caddy.Load(j, true)
	if err != nil {
		return fmt.Errorf("could not reload caddy config %v", err.Error())
	}

	return nil
}
