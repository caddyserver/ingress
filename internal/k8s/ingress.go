package k8s

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	apiv1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type IngressHandlers struct {
	AddFunc    func(obj *networkingv1.Ingress)
	UpdateFunc func(oldObj, newObj *networkingv1.Ingress)
	DeleteFunc func(obj *networkingv1.Ingress)
}

type IngressParams struct {
	InformerFactory   informers.SharedInformerFactory
	ClassName         string
	ClassNameRequired bool
}

func WatchIngresses(options IngressParams, funcs IngressHandlers) cache.SharedIndexInformer {
	informer := options.InformerFactory.Networking().V1().Ingresses().Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ingress, ok := obj.(*networkingv1.Ingress)

			if ok && isControllerIngress(options, ingress) {
				funcs.AddFunc(ingress)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldIng, ok1 := oldObj.(*networkingv1.Ingress)
			newIng, ok2 := newObj.(*networkingv1.Ingress)

			if ok1 && ok2 && isControllerIngress(options, newIng) {
				funcs.UpdateFunc(oldIng, newIng)
			}
		},
		DeleteFunc: func(obj interface{}) {
			ingress, ok := obj.(*networkingv1.Ingress)

			if ok && isControllerIngress(options, ingress) {
				funcs.DeleteFunc(ingress)
			}
		},
	})

	return informer
}

// isControllerIngress check if the ingress object can be controlled by us
func isControllerIngress(options IngressParams, ingress *networkingv1.Ingress) bool {
	ingressClass := ingress.Annotations["kubernetes.io/ingress.class"]
	if ingressClass == "" && ingress.Spec.IngressClassName != nil {
		ingressClass = *ingress.Spec.IngressClassName
	}

	if !options.ClassNameRequired && ingressClass == "" {
		return true
	}
	return ingressClass == options.ClassName
}

func UpdateIngressStatus(kubeClient *kubernetes.Clientset, ing *networkingv1.Ingress, status []apiv1.LoadBalancerIngress) (*networkingv1.Ingress, error) {
	ingClient := kubeClient.NetworkingV1().Ingresses(ing.Namespace)

	currIng, err := ingClient.Get(context.TODO(), ing.Name, v1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("unexpected error searching Ingress %v/%v", ing.Namespace, ing.Name))
	}

	currIng.Status.LoadBalancer.Ingress = status

	return ingClient.UpdateStatus(context.TODO(), currIng, v1.UpdateOptions{})
}
