package k8s

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	v12 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

// ConfigMapOptions represents global options set through a configmap
type ConfigMapOptions struct {
	Debug                     bool           `json:"debug,omitempty"`
	AcmeCA                    string         `json:"acmeCA,omitempty"`
	Email                     string         `json:"email,omitempty"`
	ProxyProtocol             bool           `json:"proxyProtocol,omitempty"`
	Metrics                   bool           `json:"metrics,omitempty"`
	OnDemandTLS               bool           `json:"onDemandTLS,omitempty"`
	OnDemandRateLimitInterval caddy.Duration `json:"onDemandTLSRateLimitInterval,omitempty"`
	OnDemandRateLimitBurst    int            `json:"onDemandTLSRateLimitBurst,omitempty"`
	OnDemandAsk               string         `json:"onDemandTLSAsk,omitempty"`
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

func WatchConfigMaps(options ConfigMapParams, funcs ConfigMapHandlers) cache.SharedIndexInformer {
	informer := options.InformerFactory.Core().V1().ConfigMaps().Informer()

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

	return informer
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
	err = decoder.Decode(cm.Data)
	if err != nil {
		return nil, errors.Wrap(err, "unexpected error parsing configmap")
	}

	return &cfgMap, nil
}
