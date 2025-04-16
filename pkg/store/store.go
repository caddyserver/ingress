package store

import (
	"go.uber.org/zap"
	apicore "k8s.io/api/core/v1"
	apinetworking "k8s.io/api/networking/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	isControlledIngressIndex string = "is-controlled-ingress"
	yesIndexValue            string = "y"
)

// Store contains resources used to generate Caddy config
type Store struct {
	Logger          *zap.SugaredLogger
	KubeClient      *kubernetes.Clientset
	Options         *Options
	ConfigMap       *ConfigMapOptions
	ConfigNamespace string
	CurrentPod      *PodInfo
	ingressCache    cache.Indexer
	secretCache     cache.Indexer
}

// NewStore returns a new store that keeps track of K8S resources needed by the controller.
func NewStore(
	logger *zap.SugaredLogger,
	kubeClient *kubernetes.Clientset,
	opts Options,
	configNamespace string,
	podInfo *PodInfo,
	ingressCache cache.Indexer,
	secretCache cache.Indexer,
) (*Store, error) {
	s := &Store{
		Logger:          logger,
		KubeClient:      kubeClient,
		Options:         &opts,
		ConfigMap:       &ConfigMapOptions{},
		ConfigNamespace: configNamespace,
		CurrentPod:      podInfo,
		ingressCache:    ingressCache,
		secretCache:     secretCache,
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
