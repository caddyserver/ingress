package k8s

import (
	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

type TLSSecretHandlers struct {
	AddFunc    func(obj *v12.Secret)
	UpdateFunc func(oldObj, newObj *v12.Secret)
	DeleteFunc func(obj *v12.Secret)
}

type TLSSecretParams struct {
	InformerFactory informers.SharedInformerFactory
}

func WatchTLSSecrets(options TLSSecretParams, funcs TLSSecretHandlers) cache.SharedIndexInformer {
	informer := options.InformerFactory.Core().V1().Secrets().Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			secret, ok := obj.(*v12.Secret)

			if ok && secret.Type == v12.SecretTypeTLS {
				funcs.AddFunc(secret)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldSecret, ok1 := oldObj.(*v12.Secret)
			newSecret, ok2 := newObj.(*v12.Secret)

			if ok1 && ok2 && newSecret.Type == v12.SecretTypeTLS {
				funcs.UpdateFunc(oldSecret, newSecret)
			}
		},
		DeleteFunc: func(obj interface{}) {
			secret, ok := obj.(*v12.Secret)

			if ok && secret.Type == v12.SecretTypeTLS {
				funcs.DeleteFunc(secret)
			}
		},
	})

	return informer
}

func ListTLSSecrets(options TLSSecretParams, ings []*v1.Ingress) ([]*v12.Secret, error) {
	lister := options.InformerFactory.Core().V1().Secrets().Lister()

	tlsSecrets := []*v12.Secret{}
	for _, ing := range ings {
		for _, tlsRule := range ing.Spec.TLS {
			secret, err := lister.Secrets(ing.Namespace).Get(tlsRule.SecretName)
			// TODO Handle errors
			if err == nil {
				tlsSecrets = append(tlsSecrets, secret)
			}
		}
	}
	return tlsSecrets, nil
}

func IsManagedTLSSecret(secret *v12.Secret, ings []*v1.Ingress) bool {
	for _, ing := range ings {
		for _, tlsRule := range ing.Spec.TLS {
			if tlsRule.SecretName == secret.Name && ing.Namespace == secret.Namespace {
				return true
			}
		}
	}
	return false
}
