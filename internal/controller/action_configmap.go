package controller

import (
	"github.com/caddyserver/ingress/pkg/store"
	v1 "k8s.io/api/core/v1"
)

// onConfigMapAdded runs when the ConfigMap is created.
func (c *CaddyController) onConfigMapAdded(obj *v1.ConfigMap) error {
	c.logger.Infof("ConfigMap created (%s/%s)", obj.Namespace, obj.Name)

	cfg, err := store.ParseConfigMap(obj)
	if err == nil {
		c.resourceStore.ConfigMap = cfg
	}
	return err
}

// onConfigMapUpdated is run when the ConfigMap is updated.
func (c *CaddyController) onConfigMapUpdated(obj *v1.ConfigMap) error {
	c.logger.Infof("ConfigMap updated (%s/%s)", obj.Namespace, obj.Name)

	cfg, err := store.ParseConfigMap(obj)
	if err == nil {
		c.resourceStore.ConfigMap = cfg
	}
	return err
}

// onConfigMapDeleted is run when the ConfigMap is deleted.
func (c *CaddyController) onConfigMapDeleted(obj *v1.ConfigMap) error {
	c.logger.Infof("ConfigMap deleted (%s/%s)", obj.Namespace, obj.Name)

	c.resourceStore.ConfigMap = nil
	return nil
}

// isOurConfigMap returns true if this is the config map with our global options
func (c *CaddyController) isOurConfigMap(obj *v1.ConfigMap) bool {
	return obj.Name == c.resourceStore.Options.ConfigMapName
}

// watchConfigMap installs event handlers for the ConfigMap containing global options
func (c *CaddyController) watchConfigMap() {
	c.informers.ConfigMap = c.factories.ConfigNamespace.Core().V1().ConfigMaps().Informer()
	c.informers.ConfigMap.SetTransform(c.configMapTransform)
	c.informers.ConfigMap.AddEventHandler(&QueuedEventHandlers[v1.ConfigMap]{
		Queue:      c.syncQueue,
		FilterFunc: c.isOurConfigMap,
		AddFunc:    c.onConfigMapAdded,
		UpdateFunc: c.onConfigMapUpdated,
		DeleteFunc: c.onConfigMapDeleted,
	})
}

// configMapTransform ensures the informer doesn't cache data we don't care about.
func (c *CaddyController) configMapTransform(obj any) (any, error) {
	if obj, ok := obj.(*v1.ConfigMap); ok {
		if !c.isOurConfigMap(obj) {
			obj.Data = nil
		}
	}
	return obj, nil
}
