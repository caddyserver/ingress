package controller

import (
	"github.com/caddyserver/ingress/pkg/k8s"
	"github.com/sirupsen/logrus"
	"gopkg.in/go-playground/pool.v3"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/api/networking/v1beta1"
	"k8s.io/client-go/kubernetes"
	"net"
	"sort"
	"strings"
)

// dispatchSync is run every syncInterval duration to sync ingress source address fields.
func (c *CaddyController) dispatchSync() {
	c.syncQueue.Add(SyncStatusAction{})
}

// SyncStatusAction provides an implementation of the action interface.
type SyncStatusAction struct {
}

// handle is run when a syncStatusAction appears in the queue.
func (r SyncStatusAction) handle(c *CaddyController) error {
	return c.syncStatus(c.resourceStore.Ingresses)
}

// syncStatus ensures that the ingress source address points to this ingress controller's IP address.
func (c *CaddyController) syncStatus(ings []*v1beta1.Ingress) error {
	addrs, err := k8s.GetAddresses(c.podInfo, c.kubeClient)
	if err != nil {
		return err
	}

	logrus.Debugf("Syncing %d Ingress resources source addresses", len(ings))
	c.updateIngStatuses(sliceToLoadBalancerIngress(addrs), ings)

	return nil
}

// updateIngStatuses starts a queue and adds all monitored ingresses to update their status source address to the on
// that the ingress controller is running on. This is called by the syncStatus queue.
func (c *CaddyController) updateIngStatuses(controllerAddresses []apiv1.LoadBalancerIngress, ings []*v1beta1.Ingress) {
	p := pool.NewLimited(10)
	defer p.Close()

	batch := p.Batch()
	sort.SliceStable(controllerAddresses, lessLoadBalancerIngress(controllerAddresses))

	for _, ing := range ings {
		curIPs := ing.Status.LoadBalancer.Ingress
		sort.SliceStable(curIPs, lessLoadBalancerIngress(curIPs))

		// check to see if ingresses source address does not match the ingress controller's.
		if ingressSliceEqual(curIPs, controllerAddresses) {
			logrus.Debugf("skipping update of Ingress %v/%v (no change)", ing.Namespace, ing.Name)
			continue
		}

		batch.Queue(runUpdate(ing, controllerAddresses, c.kubeClient))
	}

	batch.QueueComplete()
	batch.WaitAll()
}

// runUpdate updates the ingress status field.
func runUpdate(ing *v1beta1.Ingress, status []apiv1.LoadBalancerIngress, client *kubernetes.Clientset) pool.WorkFunc {
	return func(wu pool.WorkUnit) (interface{}, error) {
		if wu.IsCancelled() {
			return nil, nil
		}

		_, err := k8s.UpdateIngressStatus(client, ing, status)
		if err != nil {
			logrus.Warningf("error updating ingress rule: %v", err)
		}

		return true, nil
	}
}

// ingressSliceEqual determines if the ingress source matches the ingress controller's.
func ingressSliceEqual(lhs, rhs []apiv1.LoadBalancerIngress) bool {
	if len(lhs) != len(rhs) {
		return false
	}

	for i := range lhs {
		if lhs[i].IP != rhs[i].IP {
			return false
		}
		if lhs[i].Hostname != rhs[i].Hostname {
			return false
		}
	}

	return true
}

// lessLoadBalancerIngress is a sorting function for ingress hostnames.
func lessLoadBalancerIngress(addrs []apiv1.LoadBalancerIngress) func(int, int) bool {
	return func(a, b int) bool {
		switch strings.Compare(addrs[a].Hostname, addrs[b].Hostname) {
		case -1:
			return true
		case 1:
			return false
		}
		return addrs[a].IP < addrs[b].IP
	}
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
