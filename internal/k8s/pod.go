package k8s

import (
	"context"
	"fmt"
	"os"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// Info contains runtime information about the pod running the Ingress controller
type Info struct {
	Name      string
	Namespace string
	// Labels selectors of the running pod
	// This is used to search for other Ingress controller pods
	Labels map[string]string
}

// GetAddresses gets the ip address or name of the node in the cluster that the
// ingress controller is running on.
func GetAddresses(p *Info, kubeClient *kubernetes.Clientset) ([]string, error) {
	var addrs []string

	// Get services that may select this pod
	svcs, err := kubeClient.CoreV1().Services(p.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, svc := range svcs.Items {
		if labels.Set(p.Labels).AsSelector().Matches(labels.Set(svc.Spec.Selector)) {
			addr := GetAddressFromService(&svc)
			if addr != "" {
				addrs = append(addrs, addr)
			}
		}
	}

	return addrs, nil
}

// GetNodeIPOrName returns the IP address or the name of a node in the cluster
func GetAddressFromService(service *apiv1.Service) string {
	switch service.Spec.Type {
	case apiv1.ServiceTypeNodePort:
	case apiv1.ServiceTypeClusterIP:
		return service.Spec.ClusterIP
	case apiv1.ServiceTypeExternalName:
		return service.Spec.ExternalName
	case apiv1.ServiceTypeLoadBalancer:
		{
			if len(service.Status.LoadBalancer.Ingress) > 0 {
				ingress := service.Status.LoadBalancer.Ingress[0]
				if ingress.Hostname != "" {
					return ingress.Hostname
				}
				return ingress.IP
			}
		}
	}
	return ""
}

// GetPodDetails returns runtime information about the pod:
// name, namespace and IP of the node where it is running
func GetPodDetails(kubeClient *kubernetes.Clientset) (*Info, error) {
	podName := os.Getenv("POD_NAME")
	podNs := os.Getenv("POD_NAMESPACE")

	if podName == "" || podNs == "" {
		return nil, fmt.Errorf("unable to get POD information (missing POD_NAME or POD_NAMESPACE environment variable")
	}

	pod, _ := kubeClient.CoreV1().Pods(podNs).Get(context.TODO(), podName, metav1.GetOptions{})
	if pod == nil {
		return nil, fmt.Errorf("unable to get POD information")
	}

	return &Info{
		Name:      podName,
		Namespace: podNs,
		Labels:    pod.GetLabels(),
	}, nil
}
