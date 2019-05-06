package controller

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	pool "gopkg.in/go-playground/pool.v3"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

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
			klog.V(3).Infof("skipping update of Ingress %v/%v (no change)", ing.Namespace, ing.Name)
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

		ingClient := client.ExtensionsV1beta1().Ingresses(ing.Namespace)

		currIng, err := ingClient.Get(ing.Name, metav1.GetOptions{})
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("unexpected error searching Ingress %v/%v", ing.Namespace, ing.Name))
		}

		klog.Infof("updating Ingress %v/%v status from %v to %v", currIng.Namespace, currIng.Name, currIng.Status.LoadBalancer.Ingress, status)
		currIng.Status.LoadBalancer.Ingress = status

		_, err = ingClient.UpdateStatus(currIng)
		if err != nil {
			klog.Warningf("error updating ingress rule: %v", err)
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
