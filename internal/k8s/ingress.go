package k8s

import (
	"context"
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func UpdateIngressStatus(kubeClient *kubernetes.Clientset, ing *networkingv1.Ingress, status []networkingv1.IngressLoadBalancerIngress) (*networkingv1.Ingress, error) {
	ingClient := kubeClient.NetworkingV1().Ingresses(ing.Namespace)

	currIng, err := ingClient.Get(context.TODO(), ing.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("unexpected error searching Ingress %v/%v: %w", ing.Namespace, ing.Name, err)
	}

	currIng.Status.LoadBalancer.Ingress = status

	return ingClient.UpdateStatus(context.TODO(), currIng, metav1.UpdateOptions{})
}
