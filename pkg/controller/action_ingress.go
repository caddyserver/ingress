package controller

import (
	"github.com/sirupsen/logrus"
	"k8s.io/api/networking/v1beta1"
)

// IngressAddedAction provides an implementation of the action interface.
type IngressAddedAction struct {
	resource *v1beta1.Ingress
}

// IngressUpdatedAction provides an implementation of the action interface.
type IngressUpdatedAction struct {
	resource    *v1beta1.Ingress
	oldResource *v1beta1.Ingress
}

// IngressDeletedAction provides an implementation of the action interface.
type IngressDeletedAction struct {
	resource *v1beta1.Ingress
}

// onIngressAdded runs when an ingress resource is added to the cluster.
func (c *CaddyController) onIngressAdded(obj *v1beta1.Ingress) {
	c.syncQueue.Add(IngressAddedAction{
		resource: obj,
	})
}

// onIngressUpdated is run when an ingress resource is updated in the cluster.
func (c *CaddyController) onIngressUpdated(old *v1beta1.Ingress, new *v1beta1.Ingress) {
	c.syncQueue.Add(IngressUpdatedAction{
		resource:    new,
		oldResource: old,
	})
}

// onIngressDeleted is run when an ingress resource is deleted from the cluster.
func (c *CaddyController) onIngressDeleted(obj *v1beta1.Ingress) {
	c.syncQueue.Add(IngressDeletedAction{
		resource: obj,
	})
}

func (r IngressAddedAction) handle(c *CaddyController) error {
	logrus.Info("New ingress resource detected")

	// add this ingress to the internal store
	c.resourceStore.AddIngress(r.resource)

	// Ingress may now have a TLS config
	return c.watchTLSSecrets()
}

func (r IngressUpdatedAction) handle(c *CaddyController) error {
	logrus.Info("Ingress resource update detected")

	// add or update this ingress in the internal store
	c.resourceStore.AddIngress(r.resource)

	// Ingress may now have a TLS config
	return c.watchTLSSecrets()
}

func (r IngressDeletedAction) handle(c *CaddyController) error {
	logrus.Info("Ingress resource deletion detected")

	// delete all resources from caddy config that are associated with this resource
	c.resourceStore.PluckIngress(r.resource)
	return nil
}
