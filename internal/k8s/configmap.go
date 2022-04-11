package k8s

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

type ConfigMapHandlers struct {
	AddFunc    func(obj *v1.ConfigMap)
	UpdateFunc func(oldObj, newObj *v1.ConfigMap)
	DeleteFunc func(obj *v1.ConfigMap)
}

type ConfigMapParams struct {
	Namespace       string
	InformerFactory informers.SharedInformerFactory
	ConfigMapName   string
}

func isControllerConfigMap(cm *v1.ConfigMap, name string) bool {
	return cm.GetName() == name
}

func WatchConfigMaps(options ConfigMapParams, funcs ConfigMapHandlers) cache.SharedIndexInformer {
	informer := options.InformerFactory.Core().V1().ConfigMaps().Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			cm, ok := obj.(*v1.ConfigMap)

			if ok && isControllerConfigMap(cm, options.ConfigMapName) {
				funcs.AddFunc(cm)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldCM, ok1 := oldObj.(*v1.ConfigMap)
			newCM, ok2 := newObj.(*v1.ConfigMap)

			if ok1 && ok2 && isControllerConfigMap(newCM, options.ConfigMapName) {
				funcs.UpdateFunc(oldCM, newCM)
			}
		},
		DeleteFunc: func(obj interface{}) {
			cm, ok := obj.(*v1.ConfigMap)

			if ok && isControllerConfigMap(cm, options.ConfigMapName) {
				funcs.DeleteFunc(cm)
			}
		},
	})

	return informer
}
