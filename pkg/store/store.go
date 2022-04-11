package store

import (
	v1 "k8s.io/api/networking/v1"
)

// Store contains resources used to generate Caddy config
type Store struct {
	Options    *Options
	ConfigMap  *ConfigMapOptions
	Ingresses  []*v1.Ingress
	CurrentPod *PodInfo
}

// NewStore returns a new store that keeps track of K8S resources needed by the controller.
func NewStore(opts Options, podInfo *PodInfo) *Store {
	s := &Store{
		Options:    &opts,
		Ingresses:  []*v1.Ingress{},
		ConfigMap:  &ConfigMapOptions{},
		CurrentPod: podInfo,
	}
	return s
}

// AddIngress adds an ingress to the store. It updates the element at the given index if it is unique.
func (s *Store) AddIngress(ing *v1.Ingress) {
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
func (s *Store) PluckIngress(ing *v1.Ingress) {
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

func (s *Store) HasManagedTLS() bool {
	for _, ing := range s.Ingresses {
		if len(ing.Spec.TLS) > 0 {
			return true
		}
	}
	return false
}
