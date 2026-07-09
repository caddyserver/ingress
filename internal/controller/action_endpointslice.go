package controller

import (
	apidiscovery "k8s.io/api/discovery/v1"
)

// onEndpointSliceAdded runs when an ingress resource is added to the cluster.
func (c *CaddyController) onEndpointSliceAdded(obj *apidiscovery.EndpointSlice) error {
	c.logger.Infof("EndpointSlice created (%s/%s)", obj.Namespace, obj.Name)
	return nil
}

// onEndpointSliceUpdated is run when an ingress resource is updated in the cluster.
func (c *CaddyController) onEndpointSliceUpdated(obj *apidiscovery.EndpointSlice) error {
	c.logger.Infof("EndpointSlice updated (%s/%s)", obj.Namespace, obj.Name)
	return nil
}

// onEndpointSliceDeleted is run when an ingress resource is deleted from the cluster.
func (c *CaddyController) onEndpointSliceDeleted(obj *apidiscovery.EndpointSlice) error {
	c.logger.Infof("EndpointSlice deleted (%s/%s)", obj.Namespace, obj.Name)
	return nil
}

func (c *CaddyController) watchEndpointSlices() {
	c.informers.EndpointSlice = c.factories.WatchedNamespace.Discovery().V1().EndpointSlices().Informer()
	c.informers.EndpointSlice.AddEventHandler(&QueuedEventHandlers[apidiscovery.EndpointSlice]{
		Queue:      c.syncQueue,
		AddFunc:    c.onEndpointSliceAdded,
		UpdateFunc: c.onEndpointSliceUpdated,
		DeleteFunc: c.onEndpointSliceDeleted,
	})
}
