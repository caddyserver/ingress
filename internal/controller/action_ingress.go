package controller

import (
	v1 "k8s.io/api/networking/v1"
)

// onIngressAdded runs when an ingress resource is added to the cluster.
func (c *CaddyController) onIngressAdded(obj *v1.Ingress) error {
	c.logger.Infof("Ingress created (%s/%s)", obj.Namespace, obj.Name)
	return nil
}

// onIngressUpdated is run when an ingress resource is updated in the cluster.
func (c *CaddyController) onIngressUpdated(obj *v1.Ingress) error {
	c.logger.Infof("Ingress updated (%s/%s)", obj.Namespace, obj.Name)
	return nil
}

// onIngressDeleted is run when an ingress resource is deleted from the cluster.
func (c *CaddyController) onIngressDeleted(obj *v1.Ingress) error {
	c.logger.Infof("Ingress deleted (%s/%s)", obj.Namespace, obj.Name)
	return nil
}

// watchIngresses
func (c *CaddyController) watchIngresses() {
	c.informers.Ingress = c.factories.WatchedNamespace.Networking().V1().Ingresses().Informer()
	c.informers.Ingress.SetTransform(c.ingressTransform)
	c.informers.Ingress.AddEventHandler(&QueuedEventHandlers[v1.Ingress]{
		Queue:      c.syncQueue,
		AddFunc:    c.onIngressAdded,
		UpdateFunc: c.onIngressUpdated,
		DeleteFunc: c.onIngressDeleted,
	})
}

// ingressTransform modifies the ingress resource to fill the
// IngressClassName field if a legacy annotation is present.
func (*CaddyController) ingressTransform(obj any) (any, error) {
	if obj, ok := obj.(*v1.Ingress); ok {
		legacyAnnotation := obj.Annotations["kubernetes.io/obj.class"]
		if legacyAnnotation != "" && obj.Spec.IngressClassName == nil {
			obj.Spec.IngressClassName = &legacyAnnotation
		}
	}
	return obj, nil
}
