package caddy

import (
	"encoding/json"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddytls"
)

// StorageValues represents the config for certmagic storage providers.
type StorageValues struct {
	Namespace string `json:"namespace"`
}

// Storage represents the certmagic storage configuration.
type Storage struct {
	System string `json:"module"`
	StorageValues
}

// Config represents a caddy2 config file.
type Config struct {
	Storage Storage                `json:"storage"`
	Apps    map[string]interface{} `json:"apps"`
	Logging caddy.Logging          `json:"logging"`
}

// ControllerConfig represents ingress controller config received through cli arguments.
type ControllerConfig struct {
	WatchNamespace string
	ConfigMapName  string
}

// NewConfig returns a base plain slate caddy2 config file.
func NewConfig(namespace string, cfgMapConfig *Config) *Config {
	var cfg *Config

	if cfgMapConfig != nil {
		cfg = cfgMapConfig
	} else {
		cfg = &Config{
			Logging: caddy.Logging{},
			Apps: map[string]interface{}{
				"tls": &caddytls.TLS{
					CertificatesRaw: caddy.ModuleMap{},
				},
				"http": &caddyhttp.App{
					Servers: map[string]*caddyhttp.Server{
						"metrics_server": metricsServer(),
						"ingress_server": {
							AutoHTTPS: &caddyhttp.AutoHTTPSConfig{},
							Listen:    []string{":443"},
						},
					},
				},
			},
		}
	}

	// set cert-magic storage provider
	cfg.Storage = Storage{
		System: "secret_store",
		StorageValues: StorageValues{
			Namespace: namespace,
		},
	}

	return cfg
}

func metricsServer() *caddyhttp.Server {
	return &caddyhttp.Server{
		// TODO Make this port configurable
		Listen:    []string{":9765"},
		AutoHTTPS: &caddyhttp.AutoHTTPSConfig{Disabled: true},
		Routes: []caddyhttp.Route{{
			HandlersRaw: []json.RawMessage{json.RawMessage(`{ "handler": "metrics" }`)},
			MatcherSetsRaw: []caddy.ModuleMap{{
				"path": caddyconfig.JSON(caddyhttp.MatchPath{"/metrics"}, nil),
			}},
		}},
	}
}
