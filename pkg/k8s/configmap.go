package k8s

import (
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	v12 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

// ConfigMapOptions represents global options set through a configmap
type ConfigMapOptions struct {
	Debug  bool   `json:"debug"`
	AcmeCA string `json:"acmeCA"`
	Email  string `json:"email"`
}

type ConfigMapHandlers struct {
	AddFunc    func(obj *v12.ConfigMap)
	UpdateFunc func(oldObj, newObj *v12.ConfigMap)
	DeleteFunc func(obj *v12.ConfigMap)
}

type ConfigMapParams struct {
	Namespace       string
	InformerFactory informers.SharedInformerFactory
	ConfigMapName   string
}

func isControllerConfigMap(cm *v12.ConfigMap, name string) bool {
	return cm.GetName() == name
}

func WatchConfigMaps(options ConfigMapParams, funcs ConfigMapHandlers) (cache.SharedIndexInformer, cache.Store) {
	informer := options.InformerFactory.Core().V1().ConfigMaps().Informer()
	store := informer.GetStore()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			cm, ok := obj.(*v12.ConfigMap)

			if ok && isControllerConfigMap(cm, options.ConfigMapName) {
				funcs.AddFunc(cm)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldCM, ok1 := oldObj.(*v12.ConfigMap)
			newCM, ok2 := newObj.(*v12.ConfigMap)

			if ok1 && ok2 && isControllerConfigMap(newCM, options.ConfigMapName) {
				funcs.UpdateFunc(oldCM, newCM)
			}
		},
		DeleteFunc: func(obj interface{}) {
			cm, ok := obj.(*v12.ConfigMap)

			if ok && isControllerConfigMap(cm, options.ConfigMapName) {
				funcs.DeleteFunc(cm)
			}
		},
	})

	return informer, store
}

func GetConfigMapOptions(opts ConfigMapParams) (*ConfigMapOptions, error) {
	cm, err := opts.InformerFactory.Core().V1().ConfigMaps().Lister().ConfigMaps("").Get(opts.ConfigMapName)

	if err != nil {
		return nil, errors.Wrap(err, "could not get option configmap")
	}

	return ParseConfigMap(cm)
}

func ParseConfigMap(cm *v12.ConfigMap) (*ConfigMapOptions, error) {
	// parse configmap
	cfgMap := ConfigMapOptions{}
	config := &mapstructure.DecoderConfig{
		Metadata:         nil,
		WeaklyTypedInput: true,
		Result:           &cfgMap,
		TagName:          "json",
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return nil, errors.Wrap(err, "unexpected error creating decoder")
	}
	err = decoder.Decode(cm)
	if err != nil {
		return nil, errors.Wrap(err, "unexpected error parsing configmap")
	}

	return &cfgMap, nil
}
