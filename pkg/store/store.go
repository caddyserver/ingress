package store

import (
	apinetworking "k8s.io/api/networking/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	isControlledIngressIndex string = "is-controlled-ingress"
	yesIndexValue            string = "y"
)

// Store contains resources used to generate Caddy config
type Store struct {
	Options         *Options
	ConfigMap       *ConfigMapOptions
	ConfigNamespace string
	CurrentPod      *PodInfo
	ingressCache    cache.Indexer
}

// NewStore returns a new store that keeps track of K8S resources needed by the controller.
func NewStore(
	opts Options,
	configNamespace string,
	podInfo *PodInfo,
	ingressCache cache.Indexer,
) (*Store, error) {
	s := &Store{
		Options:         &opts,
		ConfigMap:       &ConfigMapOptions{},
		ConfigNamespace: configNamespace,
		CurrentPod:      podInfo,
		ingressCache:    ingressCache,
	}

	err := ingressCache.AddIndexers(map[string]cache.IndexFunc{
		// Contains the set of ingresses we control.
		isControlledIngressIndex: func(obj any) ([]string, error) {
			ingressClass := obj.(*apinetworking.Ingress).Spec.IngressClassName
			if (ingressClass != nil && *ingressClass == opts.ClassName) ||
				(ingressClass == nil && !opts.ClassNameRequired) {
				return []string{yesIndexValue}, nil
			}
			return nil, nil
		},
	})

	return s, err
}

// Ingresses returns a list of Ingress resources that match our IngressClass.
func (s *Store) Ingresses() []*apinetworking.Ingress {
	// Note: errors only when the index does not exist.
	list, _ := s.ingressCache.ByIndex(isControlledIngressIndex, yesIndexValue)

	result := make([]*apinetworking.Ingress, 0, len(list))
	for _, ingress := range list {
		result = append(result, ingress.(*apinetworking.Ingress))
	}
	return result
}

func (s *Store) HasManagedTLS() bool {
	for _, ing := range s.Ingresses() {
		if len(ing.Spec.TLS) > 0 {
			return true
		}
	}
	return false
}
