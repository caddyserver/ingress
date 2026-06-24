package k8s

import (
	v1 "k8s.io/api/core/v1"
)

type ConfigMapFilter struct {
	ConfigMapName string
}

func (cmf *ConfigMapFilter) Filter(obj *v1.ConfigMap) bool {
	return obj.GetName() == cmf.ConfigMapName
}
