package store

// PodInfo contains runtime information about the pod running the Ingress controller
type PodInfo struct {
	Name      string
	Namespace string
	// Labels selectors of the running pod
	// This is used to search for other Ingress controller pods
	Labels map[string]string
}
