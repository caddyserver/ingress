package controller

import (
	apiv1 "k8s.io/api/core/v1"
)

// watchSecrets starts listening for changes to secrets.
func (c *CaddyController) watchSecrets() {
	c.informers.Secret = c.factories.WatchedNamespace.Core().V1().Secrets().Informer()
	c.informers.Secret.SetTransform(c.secretTransform)
	c.informers.Secret.AddEventHandler(&QueuedEventHandlers[apiv1.Secret]{
		Queue: c.syncQueue,
	})
}

// secretTransform strips secrets of all contents, so we only store metadata.
// This prevents us from storing secrets in memory that are none of our business.
func (*CaddyController) secretTransform(obj any) (any, error) {
	if obj, ok := obj.(*apiv1.Secret); ok {
		// TODO: Can we secure erase secret data from memory?
		obj.Data = nil
	}
	return obj, nil
}
