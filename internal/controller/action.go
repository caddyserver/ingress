package controller

import (
	"fmt"

	"k8s.io/api/extensions/v1beta1"
)

func (c *CaddyController) onResourceAdded(obj interface{}) {
	c.syncQueue.Add(ResourceAddedAction{
		resource: obj,
	})
}

func (c *CaddyController) onResourceUpdated(old interface{}, new interface{}) {
	c.syncQueue.Add(ResourceUpdatedAction{
		resource:    new,
		oldResource: old,
	})
}

func (c *CaddyController) onResourceDeleted(obj interface{}) {
	c.syncQueue.Add(ResourceDeletedAction{
		resource: obj,
	})
}

func (c *CaddyController) onSyncStatus(obj interface{}) {
	c.syncQueue.Add(SyncStatusAction{})
}

// Action is an interface for ingress actions
type Action interface {
	handle(c *CaddyController) error
}

// ResourceAddedAction provides an implementation of the action interface
type ResourceAddedAction struct {
	resource interface{}
}

// ResourceUpdatedAction provides an implementation of the action interface
type ResourceUpdatedAction struct {
	resource    interface{}
	oldResource interface{}
}

// ResourceDeletedAction provides an implementation of the action interface
type ResourceDeletedAction struct {
	resource interface{}
}

func (r ResourceAddedAction) handle(c *CaddyController) error {
	// configure caddy to handle this resource
	ing, ok := r.resource.(*v1beta1.Ingress)
	if !ok {
		return fmt.Errorf("ResourceAddedAction: incoming resource is not of type ingress")
	}

	// 1. Parse ingress resource and convert to obj to configure caddy with
	// 2. Get current caddy config for rollback purposes
	// 3. Update internal caddy config
	// 4. Get ingress controller publish address
	// 5. call syncIngress for this specific resource
	// 6. Add this ingress to resource store

	c.resourceStore.AddIngress(ing)

	// ~~~~
	// when updating caddy config the ingress controller should bypass kube-proxy and get the ip address of
	// the pod that the deployment we are proxying to is running on so that we can proxy to that ip address port.
	// this is good for session affinity and increases performance (since we don't have to hit dns).

	// example getting an ingress
	// ingClient := c.kubeClient.ExtensionsV1beta1().Ingresses(c.namespace) // get a client to update the ingress
	// ingClient.UpdateStatus(ing) // pass an ingress with the status.address field updated
	// ~~~

	return nil
}

func (r ResourceUpdatedAction) handle(c *CaddyController) error {
	// find the caddy config related to the oldResource and update it

	fmt.Printf("\nUpdated resource:\n +%v\n\nOld resource: \n %+v\n", r.resource, r.oldResource)

	return nil
}

func (r ResourceDeletedAction) handle(c *CaddyController) error {
	// delete all resources from caddy config that are associated with this resource
	// reload caddy config

	fmt.Printf("\nDeleted resource:\n +%v\n", r.resource)

	return nil
}
