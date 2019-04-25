package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"bitbucket.org/lightcodelabs/caddy2"
	"bitbucket.org/lightcodelabs/ingress/internal/pod"
	"bitbucket.org/lightcodelabs/ingress/internal/store"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/fields"
	run "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	// load required caddy plugins
	_ "bitbucket.org/lightcodelabs/caddy2/modules/caddyhttp"
	_ "bitbucket.org/lightcodelabs/caddy2/modules/caddyhttp/caddylog"
	_ "bitbucket.org/lightcodelabs/caddy2/modules/caddyhttp/staticfiles"
	_ "bitbucket.org/lightcodelabs/proxy"
)

// ResourceMap are resources from where changes are going to be detected
var ResourceMap = map[string]run.Object{
	"ingresses": &v1beta1.Ingress{},
}

const (
	// how often we should attempt to keep ingress resource's source address in sync
	syncInterval = time.Second * 30
)

// CaddyController represents an caddy ingress controller.
type CaddyController struct {
	resourceStore *store.Store
	kubeClient    *kubernetes.Clientset
	namespace     string
	indexer       cache.Indexer
	syncQueue     workqueue.RateLimitingInterface
	statusQueue   workqueue.RateLimitingInterface // statusQueue performs ingress status updates every 60 seconds but inserts the work into the sync queue
	informer      cache.Controller
	podInfo       *pod.Info
}

// NewCaddyController returns an instance of the caddy ingress controller.
func NewCaddyController(namespace string, kubeClient *kubernetes.Clientset, resource string, restClient rest.Interface) *CaddyController {
	controller := &CaddyController{
		kubeClient:  kubeClient,
		namespace:   namespace,
		syncQueue:   workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		statusQueue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}

	ingressListWatcher := cache.NewListWatchFromClient(restClient, resource, namespace, fields.Everything())
	indexer, informer := cache.NewIndexerInformer(ingressListWatcher, ResourceMap[resource], 0, cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.onResourceAdded,
		UpdateFunc: controller.onResourceUpdated,
		DeleteFunc: controller.onResourceDeleted,
	}, cache.Indexers{})

	controller.indexer = indexer
	controller.informer = informer
	controller.resourceStore = store.NewStore(controller.kubeClient)

	podInfo, err := pod.GetPodDetails(kubeClient)
	if err != nil {
		klog.Fatalf("Unexpected error obtaining pod information: %v", err)
	}
	controller.podInfo = podInfo

	// attempt to do initial sync with ingresses
	controller.syncQueue.Add(SyncStatusAction{})

	return controller
}

// Shutdown stops the caddy controller.
func (c *CaddyController) Shutdown() error {
	// remove this ingress controller's ip from ingress resources.
	c.updateIngStatuses([]apiv1.LoadBalancerIngress{apiv1.LoadBalancerIngress{}}, c.resourceStore.Ingresses)
	return nil
}

// handleErrs reports errors received from queue actions.
func (c *CaddyController) handleErr(err error, action interface{}) {
	klog.Error(err)
}

func (c *CaddyController) reloadCaddy() error {
	j, err := json.Marshal(c.resourceStore.CaddyConfig)
	if err != nil {
		return err
	}

	cfgReader := bytes.NewReader(j)
	err = caddy2.Load(cfgReader)
	if err != nil {
		return err
	}

	return nil
}

// Run method starts the ingress controller.
func (c *CaddyController) Run(stopCh chan struct{}) {
	err := c.reloadCaddy()
	if err != nil {
		klog.Errorf("initial caddy config load failed, %v", err.Error())
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
	klog.Info("stopping ingress controller")

	var exitCode int
	err = c.Shutdown()
	if err != nil {
		klog.Errorf("could not shutdown ingress controller properly, %v", err.Error())
		exitCode = 1
	}

	os.Exit(exitCode)
}

// process items in the event queue
func (c *CaddyController) runWorker() {
	for c.processNextItem() {
	}
}

// if there is an ingress item in the event queue process it
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
