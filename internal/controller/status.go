package controller

import (
	"net"
	"sort"

	"bitbucket.org/lightcodelabs/ingress/internal/pod"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/klog"
)

// dispatchSync is run every syncInterval duration to sync ingress source address fields.
func (c *CaddyController) dispatchSync() {
	c.syncQueue.Add(SyncStatusAction{})
}

// SyncStatusAction provides an implementation of the action interface
type SyncStatusAction struct {
}

func (r SyncStatusAction) handle(c *CaddyController) error {
	return c.syncStatus(c.resourceStore.Ingresses)
}

// syncStatus ensures that the ingress source address points to this ingress controller's IP address.
func (c *CaddyController) syncStatus(ings []*v1beta1.Ingress) error {
	addrs, err := pod.GetAddresses(c.podInfo, c.kubeClient)
	if err != nil {
		return err
	}

	klog.Info("Synching Ingress resource source addresses")
	c.updateIngStatuses(sliceToLoadBalancerIngress(addrs), ings)

	return nil
}

// sliceToLoadBalancerIngress converts a slice of IP and/or hostnames to LoadBalancerIngress
func sliceToLoadBalancerIngress(endpoints []string) []apiv1.LoadBalancerIngress {
	lbi := []apiv1.LoadBalancerIngress{}
	for _, ep := range endpoints {
		if net.ParseIP(ep) == nil {
			lbi = append(lbi, apiv1.LoadBalancerIngress{Hostname: ep})
		} else {
			lbi = append(lbi, apiv1.LoadBalancerIngress{IP: ep})
		}
	}

	sort.SliceStable(lbi, func(a, b int) bool {
		return lbi[a].IP < lbi[b].IP
	})

	return lbi
}
