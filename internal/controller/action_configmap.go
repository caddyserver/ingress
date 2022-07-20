package controller

import (
	"github.com/caddyserver/ingress/pkg/store"
	v1 "k8s.io/api/core/v1"
)

// ConfigMapAddedAction provides an implementation of the action interface.
type ConfigMapAddedAction struct {
	resource *v1.ConfigMap
}

// ConfigMapUpdatedAction provides an implementation of the action interface.
type ConfigMapUpdatedAction struct {
	resource    *v1.ConfigMap
	oldResource *v1.ConfigMap
}

// ConfigMapDeletedAction provides an implementation of the action interface.
type ConfigMapDeletedAction struct {
	resource *v1.ConfigMap
}

// onConfigMapAdded runs when a configmap is added to the namespace.
func (c *CaddyController) onConfigMapAdded(obj *v1.ConfigMap) {
	c.syncQueue.Add(ConfigMapAddedAction{
		resource: obj,
	})
}

// onConfigMapUpdated is run when a configmap is updated in the namespace.
func (c *CaddyController) onConfigMapUpdated(old *v1.ConfigMap, new *v1.ConfigMap) {
	c.syncQueue.Add(ConfigMapUpdatedAction{
		resource:    new,
		oldResource: old,
	})
}

// onConfigMapDeleted is run when a configmap is deleted from the namespace.
func (c *CaddyController) onConfigMapDeleted(obj *v1.ConfigMap) {
	c.syncQueue.Add(ConfigMapDeletedAction{
		resource: obj,
	})
}

func (r ConfigMapAddedAction) handle(c *CaddyController) error {
	c.logger.Infof("ConfigMap created (%s/%s)", r.resource.Namespace, r.resource.Name)

	cfg, err := store.ParseConfigMap(r.resource)
	if err == nil {
		c.resourceStore.ConfigMap = cfg
	}
	return err
}

func (r ConfigMapUpdatedAction) handle(c *CaddyController) error {
	c.logger.Infof("ConfigMap updated (%s/%s)", r.resource.Namespace, r.resource.Name)

	cfg, err := store.ParseConfigMap(r.resource)
	if err == nil {
		c.resourceStore.ConfigMap = cfg
	}
	return err
}

func (r ConfigMapDeletedAction) handle(c *CaddyController) error {
	c.logger.Infof("ConfigMap deleted (%s/%s)", r.resource.Namespace, r.resource.Name)

	c.resourceStore.ConfigMap = nil
	return nil
}
