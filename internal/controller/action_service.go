package controller

import (
	apicore "k8s.io/api/core/v1"
)

// onServiceAdded runs when an serbice resource is added to the cluster.
func (c *CaddyController) onServiceAdded(obj *apicore.Service) error {
	c.logger.Infof("Service created (%s/%s)", obj.Namespace, obj.Name)
	return nil
}

// onServiceUpdated is run when an serbice resource is updated in the cluster.
func (c *CaddyController) onServiceUpdated(obj *apicore.Service) error {
	c.logger.Infof("Service updated (%s/%s)", obj.Namespace, obj.Name)
	return nil
}

// onServiceDeleted is run when an serbice resource is deleted from the cluster.
func (c *CaddyController) onServiceDeleted(obj *apicore.Service) error {
	c.logger.Infof("Service deleted (%s/%s)", obj.Namespace, obj.Name)
	return nil
}

func (c *CaddyController) watchServices() {
	c.informers.Service = c.factories.WatchedNamespace.Core().V1().Services().Informer()
	c.informers.Service.AddEventHandler(&QueuedEventHandlers[apicore.Service]{
		Queue:      c.syncQueue,
		AddFunc:    c.onServiceAdded,
		UpdateFunc: c.onServiceUpdated,
		DeleteFunc: c.onServiceDeleted,
	})
}
