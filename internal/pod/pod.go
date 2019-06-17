package pod

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/kubelet/util/sliceutils"
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
	addrs := []string{}

	// get information about all the pods running the ingress controller
	pods, err := kubeClient.CoreV1().Pods(p.Namespace).List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(p.Labels).String(),
	})
	if err != nil {
		return nil, err
	}

	for _, pod := range pods.Items {
		// only Running pods are valid
		if pod.Status.Phase != apiv1.PodRunning {
			continue
		}

		name := GetNodeIPOrName(kubeClient, pod.Spec.NodeName, true)
		if !sliceutils.StringInSlice(name, addrs) {
			addrs = append(addrs, name)
		}
	}

	return addrs, nil
}

// GetNodeIPOrName returns the IP address or the name of a node in the cluster
func GetNodeIPOrName(kubeClient *kubernetes.Clientset, name string, useInternalIP bool) string {
	node, err := kubeClient.CoreV1().Nodes().Get(name, metav1.GetOptions{})
	if err != nil {
		logrus.Errorf("Error getting node %v: %v", name, err)
		return ""
	}

	if useInternalIP {
		for _, address := range node.Status.Addresses {
			if address.Type == apiv1.NodeInternalIP {
				if address.Address != "" {
					return address.Address
				}
			}
		}
	}

	for _, address := range node.Status.Addresses {
		if address.Type == apiv1.NodeExternalIP {
			if address.Address != "" {
				return address.Address
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

	pod, _ := kubeClient.CoreV1().Pods(podNs).Get(podName, metav1.GetOptions{})
	if pod == nil {
		return nil, fmt.Errorf("unable to get POD information")
	}

	return &Info{
		Name:      podName,
		Namespace: podNs,
		Labels:    pod.GetLabels(),
	}, nil
}
