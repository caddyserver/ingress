package store

import (
	c "github.com/caddyserver/ingress/internal/caddy"
	"github.com/sirupsen/logrus"
	"k8s.io/api/networking/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Store represents a collection of ingresses and secrets that we are monitoring.
type Store struct {
	Ingresses   []*v1beta1.Ingress
	Secrets     []interface{} // TODO :- should we store the secrets in the ingress object?
	CaddyConfig *c.Config
}

// NewStore returns a new store that keeps track of ingresses and secrets. It will attempt to get
// all current ingresses before returning.
func NewStore(kubeClient *kubernetes.Clientset, namespace string, cfg c.ControllerConfig, cfgMapConfig *c.Config) *Store {
	ingresses, err := kubeClient.NetworkingV1beta1().Ingresses("").List(v1.ListOptions{})
	if err != nil {
		logrus.Errorf("could not get existing ingresses in cluster")
		return &Store{}
	}

	s := &Store{
		Ingresses: []*v1beta1.Ingress{},
	}

	for _, i := range ingresses.Items {
		s.Ingresses = append(s.Ingresses, &i)
	}

	// not using cfg map to configure the ingress controller
	if cfgMapConfig == nil {
		s.CaddyConfig = c.NewConfig(namespace, cfg)
		return s
	}

	// set cert-magic storage provider
	cfgMapConfig.Storage = c.Storage{
		System: "secret_store",
		StorageValues: c.StorageValues{
			Namespace: namespace,
		},
	}

	s.CaddyConfig = cfgMapConfig

	return s
}

// AddIngress adds an ingress to the store. It updates the element at the given index if it is unique.
func (s *Store) AddIngress(ing *v1beta1.Ingress) {
	isUniq := true

	for i := range s.Ingresses {
		in := s.Ingresses[i]
		if in.GetUID() == ing.GetUID() {
			isUniq = false
			s.Ingresses[i] = ing
		}
	}

	if isUniq {
		s.Ingresses = append(s.Ingresses, ing)
	}
}

// PluckIngress removes the ingress passed in as an argument from the stores list of ingresses.
func (s *Store) PluckIngress(ing *v1beta1.Ingress) {
	id := ing.GetUID()

	var index int
	var hasMatch bool
	for i := range s.Ingresses {
		if s.Ingresses[i].GetUID() == id {
			index = i
			hasMatch = true
			break
		}
	}

	// since order is not important we can swap the element to delete with the one at the end of the slice
	// and then set ingresses to the n-1 first elements
	if hasMatch {
		s.Ingresses[len(s.Ingresses)-1], s.Ingresses[index] = s.Ingresses[index], s.Ingresses[len(s.Ingresses)-1]
		s.Ingresses = s.Ingresses[:len(s.Ingresses)-1]
	}
}
