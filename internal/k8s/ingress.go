package k8s

import (
	"context"
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func UpdateIngressStatus(kubeClient *kubernetes.Clientset, ing *networkingv1.Ingress, status []networkingv1.IngressLoadBalancerIngress) (*networkingv1.Ingress, error) {
	ingClient := kubeClient.NetworkingV1().Ingresses(ing.Namespace)

	currIng, err := ingClient.Get(context.TODO(), ing.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("unexpected error searching Ingress %v/%v: %w", ing.Namespace, ing.Name, err)
	}

	currIng.Status.LoadBalancer.Ingress = status

	return ingClient.UpdateStatus(context.TODO(), currIng, metav1.UpdateOptions{})
}
