package store

import (
	"fmt"
	"reflect"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/mitchellh/mapstructure"
	apiv1 "k8s.io/api/core/v1"
)

// ConfigMapOptions represents global options set through a configmap
type ConfigMapOptions struct {
	Debug                     bool           `json:"debug,omitempty"`
	AcmeCA                    string         `json:"acmeCA,omitempty"`
	AcmeEABKeyId              string         `json:"acmeEABKeyId,omitempty"`
	AcmeEABMacKey             string         `json:"acmeEABMacKey,omitempty"`
	Email                     string         `json:"email,omitempty"`
	ExperimentalSmartSort     bool           `json:"experimentalSmartSort,omitempty"`
	ProxyProtocol             bool           `json:"proxyProtocol,omitempty"`
	Metrics                   bool           `json:"metrics,omitempty"`
	OnDemandTLS               bool           `json:"onDemandTLS,omitempty"`
	OnDemandRateLimitInterval caddy.Duration `json:"onDemandRateLimitInterval,omitempty"`
	OnDemandRateLimitBurst    int            `json:"onDemandRateLimitBurst,omitempty"`
	OnDemandAsk               string         `json:"onDemandAsk,omitempty"`
	OCSPCheckInterval         caddy.Duration `json:"ocspCheckInterval,omitempty"`
}

func stringToCaddyDurationHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(caddy.Duration(time.Second)) {
			return data, nil
		}
		return caddy.ParseDuration(data.(string))
	}
}

func ParseConfigMap(cm *apiv1.ConfigMap) (*ConfigMapOptions, error) {
	// parse configmap
	cfgMap := ConfigMapOptions{}
	config := &mapstructure.DecoderConfig{
		Metadata:         nil,
		WeaklyTypedInput: true,
		Result:           &cfgMap,
		TagName:          "json",
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			stringToCaddyDurationHookFunc(),
		),
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return nil, fmt.Errorf("unexpected error creating decoder: %w", err)
	}
	err = decoder.Decode(cm.Data)
	if err != nil {
		return nil, fmt.Errorf("unexpected error parsing configmap: %w", err)
	}

	return &cfgMap, nil
}
