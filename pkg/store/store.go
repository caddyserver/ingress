package store

import (
	"fmt"

	"go.uber.org/zap"
	apicore "k8s.io/api/core/v1"
	apidiscovery "k8s.io/api/discovery/v1"
	apinetworking "k8s.io/api/networking/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	isControlledIngressIndex         string = "is-controlled-ingress"
	endpointSlicesByServiceNameIndex string = "by-service-name"
	yesIndexValue                    string = "y"
)

// Store contains resources used to generate Caddy config
type Store struct {
	Logger             *zap.SugaredLogger
	KubeClient         *kubernetes.Clientset
	Options            *Options
	ConfigMap          *ConfigMapOptions
	ConfigNamespace    string
	CurrentPod         *PodInfo
	ingressCache       cache.Indexer
	serviceCache       cache.Indexer
	endpointSliceCache cache.Indexer
	secretCache        cache.Indexer
}

// NewStore returns a new store that keeps track of K8S resources needed by the controller.
func NewStore(
	logger *zap.SugaredLogger,
	kubeClient *kubernetes.Clientset,
	opts Options,
	configNamespace string,
	podInfo *PodInfo,
	ingressCache cache.Indexer,
	serviceCache cache.Indexer,
	endpointSliceCache cache.Indexer,
	secretCache cache.Indexer,
) (*Store, error) {
	// For testing purposes, we allow these to be nil.
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	if ingressCache == nil {
		ingressCache = cache.NewIndexer(cache.MetaNamespaceKeyFunc, make(cache.Indexers))
	}
	if serviceCache == nil {
		serviceCache = cache.NewIndexer(cache.MetaNamespaceKeyFunc, make(cache.Indexers))
	}
	if endpointSliceCache == nil {
		endpointSliceCache = cache.NewIndexer(cache.MetaNamespaceKeyFunc, make(cache.Indexers))
	}
	if secretCache == nil {
		secretCache = cache.NewIndexer(cache.MetaNamespaceKeyFunc, make(cache.Indexers))
	}

	s := &Store{
		Logger:             logger,
		KubeClient:         kubeClient,
		Options:            &opts,
		ConfigMap:          &ConfigMapOptions{},
		ConfigNamespace:    configNamespace,
		CurrentPod:         podInfo,
		ingressCache:       ingressCache,
		serviceCache:       serviceCache,
		endpointSliceCache: endpointSliceCache,
		secretCache:        secretCache,
	}

	if err := ingressCache.AddIndexers(map[string]cache.IndexFunc{
		// Contains the set of ingresses we control.
		isControlledIngressIndex: func(obj any) ([]string, error) {
			ingressClass := obj.(*apinetworking.Ingress).Spec.IngressClassName
			if (ingressClass != nil && *ingressClass == opts.ClassName) ||
				(ingressClass == nil && !opts.ClassNameRequired) {
				return []string{yesIndexValue}, nil
			}
			return nil, nil
		},
	}); err != nil {
		return nil, err
	}

	if err := endpointSliceCache.AddIndexers(map[string]cache.IndexFunc{
		// Indexes endpoint slices by service name.
		endpointSlicesByServiceNameIndex: func(obj any) ([]string, error) {
			es := obj.(*apidiscovery.EndpointSlice)
			if serviceName, ok := es.Labels[apidiscovery.LabelServiceName]; ok {
				return []string{fmt.Sprintf("%s/%s", es.Namespace, serviceName)}, nil
			}
			return nil, nil
		},
	}); err != nil {
		return nil, err
	}

	return s, nil
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

// EndpointSlicesByService returns a list of endpoint slices for the given service.
// The serviceName must be specified in full as `<namespace>/<name>`
func (s *Store) EndpointSlicesByService(serviceName string) []*apidiscovery.EndpointSlice {
	// Note: errors only when the index does not exist.
	list, _ := s.endpointSliceCache.ByIndex(endpointSlicesByServiceNameIndex, serviceName)

	result := make([]*apidiscovery.EndpointSlice, 0, len(list))
	for _, obj := range list {
		result = append(result, obj.(*apidiscovery.EndpointSlice))
	}
	return result
}

// Service returns the current state of a service by name.
// The serviceName must be specified in full as `<namespace>/<name>`
func (s *Store) Service(serviceName string) *apicore.Service {
	if service, exists, _ := s.serviceCache.GetByKey(serviceName); exists {
		if service, ok := service.(*apicore.Service); ok {
			return service
		}
	}
	return nil
}

// SecretMeta returns metadata for a secret.
// The secretName must be specified in full as `<namespace>/<name>`
func (s *Store) SecretMeta(secretName string) *apicore.Secret {
	if secret, exists, _ := s.secretCache.GetByKey(secretName); exists {
		if secret, ok := secret.(*apicore.Secret); ok {
			return secret
		}
	}
	return nil
}
