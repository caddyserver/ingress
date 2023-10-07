package controller

import (
	v1 "k8s.io/api/networking/v1"
)

// IngressAddedAction provides an implementation of the action interface.
type IngressAddedAction struct {
	resource *v1.Ingress
}

// IngressUpdatedAction provides an implementation of the action interface.
type IngressUpdatedAction struct {
	resource    *v1.Ingress
	oldResource *v1.Ingress
}

// IngressDeletedAction provides an implementation of the action interface.
type IngressDeletedAction struct {
	resource *v1.Ingress
}

// onIngressAdded runs when an ingress resource is added to the cluster.
func (c *CaddyController) onIngressAdded(obj *v1.Ingress) {
	c.syncQueue.Add(IngressAddedAction{
		resource: obj,
	})
}

// onIngressUpdated is run when an ingress resource is updated in the cluster.
func (c *CaddyController) onIngressUpdated(old *v1.Ingress, new *v1.Ingress) {
	c.syncQueue.Add(IngressUpdatedAction{
		resource:    new,
		oldResource: old,
	})
}

// onIngressDeleted is run when an ingress resource is deleted from the cluster.
func (c *CaddyController) onIngressDeleted(obj *v1.Ingress) {
	c.syncQueue.Add(IngressDeletedAction{
		resource: obj,
	})
}

func (r IngressAddedAction) handle(c *CaddyController) error {
	c.logger.Infof("Ingress created (%s/%s)", r.resource.Namespace, r.resource.Name)
	// add this ingress to the internal store
	c.resourceStore.AddIngress(r.resource)

	// Ingress may now have a TLS config
	return c.watchTLSSecrets()
}

func (r IngressUpdatedAction) handle(c *CaddyController) error {
	c.logger.Infof("Ingress updated (%s/%s)", r.resource.Namespace, r.resource.Name)

	// add or update this ingress in the internal store
	c.resourceStore.AddIngress(r.resource)

	// Ingress may now have a TLS config
	return c.watchTLSSecrets()
}

func (r IngressDeletedAction) handle(c *CaddyController) error {
	c.logger.Infof("Ingress deleted (%s/%s)", r.resource.Namespace, r.resource.Name)

	// delete all resources from caddy config that are associated with this resource
	c.resourceStore.PluckIngress(r.resource)
	return nil
}
