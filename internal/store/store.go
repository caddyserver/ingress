package store

import (
	"bitbucket.org/lightcodelabs/ingress/internal/caddy"
	"k8s.io/api/extensions/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

// Store represents a collection of ingresses and secrets that we are monitoring.
type Store struct {
	Ingresses   []*v1beta1.Ingress
	Secrets     []interface{} // TODO :- should we store the secrets in the ingress object?
	CaddyConfig *caddy.Config
}

// NewStore returns a new store that keeps track of ingresses and secrets. It will attempt to get
// all current ingresses before returning.
func NewStore(kubeClient *kubernetes.Clientset) *Store {
	ingresses, err := kubeClient.ExtensionsV1beta1().Ingresses("").List(v1.ListOptions{})
	if err != nil {
		klog.Errorf("could not get existing ingresses in cluster")
		return &Store{}
	}

	s := &Store{
		Ingresses:   []*v1beta1.Ingress{},
		CaddyConfig: caddy.NewConfig(),
	}

	for _, i := range ingresses.Items {
		s.Ingresses = append(s.Ingresses, &i)
	}

	return s
}

// AddIngress adds an ingress to the store
func (s *Store) AddIngress(ing *v1beta1.Ingress) {
	isUniq := true

	for _, i := range s.Ingresses {
		if i.GetUID() == ing.GetUID() {
			isUniq = false
		}
	}

	if isUniq {
		s.Ingresses = append(s.Ingresses, ing)
	}
}
