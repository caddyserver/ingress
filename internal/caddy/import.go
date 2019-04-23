package caddy

import (
	"k8s.io/api/extensions/v1beta1"
)

// AddIngressConfig attempts to configure caddy2 for a new ingress resource.
func AddIngressConfig(c *Config, ing *v1beta1.Ingress) (*Config, error) {
	return nil, nil
}

// UpdateIngressConfig attempts to update caddy2 config for an ingress resource that has already been configured.
func UpdateIngressConfig(c *Config, ing *v1beta1.Ingress) (*Config, error) {
	return nil, nil
}

// DeleteIngressConfig attempts to update caddy2 config to remove an ingress resource.
func DeleteIngressConfig(c *Config, ing *v1beta1.Ingress) (*Config, error) {
	return nil, nil
}
