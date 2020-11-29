package caddy

import (
	"encoding/json"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddytls"
	"github.com/caddyserver/ingress/internal/controller"
)

// StorageValues represents the config for certmagic storage providers.
type StorageValues struct {
	Namespace string `json:"namespace"`
	LeaseId   string `json:"leaseId"`
}

// Storage represents the certmagic storage configuration.
type Storage struct {
	System string `json:"module"`
	StorageValues
}

// Config represents a caddy2 config file.
type Config struct {
	Admin   caddy.AdminConfig      `json:"admin,omitempty"`
	Storage Storage                `json:"storage"`
	Apps    map[string]interface{} `json:"apps"`
	Logging caddy.Logging          `json:"logging"`
}

type Converter struct{}

const (
	HttpServer    = "ingress_server"
	MetricsServer = "metrics_server"
)

func metricsServer(enabled bool) *caddyhttp.Server {
	handler := json.RawMessage(`{ "handler": "static_response" }`)
	if enabled {
		handler = json.RawMessage(`{ "handler": "metrics" }`)
	}

	return &caddyhttp.Server{
		Listen:    []string{":9765"},
		AutoHTTPS: &caddyhttp.AutoHTTPSConfig{Disabled: true},
		Routes: []caddyhttp.Route{{
			HandlersRaw: []json.RawMessage{handler},
			MatcherSetsRaw: []caddy.ModuleMap{{
				"path": caddyconfig.JSON(caddyhttp.MatchPath{"/metrics"}, nil),
			}},
		}},
	}
}

func newConfig(namespace string, store *controller.Store) (*Config, error) {
	cfg := &Config{
		Logging: caddy.Logging{},
		Apps: map[string]interface{}{
			"tls": &caddytls.TLS{
				CertificatesRaw: caddy.ModuleMap{},
			},
			"http": &caddyhttp.App{
				Servers: map[string]*caddyhttp.Server{
					MetricsServer: metricsServer(store.ConfigMap.Metrics),
					HttpServer: {
						AutoHTTPS: &caddyhttp.AutoHTTPSConfig{},
						Listen:    []string{":443"},
					},
				},
			},
		},
		Storage: Storage{
			System: "secret_store",
			StorageValues: StorageValues{
				Namespace: namespace,
				LeaseId:   store.Options.LeaseId,
			},
		},
	}

	return cfg, nil
}

func (c Converter) ConvertToCaddyConfig(namespace string, store *controller.Store) (interface{}, error) {
	cfg, err := newConfig(namespace, store)

	err = LoadIngressConfig(cfg, store)
	if err != nil {
		return cfg, err
	}

	err = LoadConfigMapOptions(cfg, store)
	if err != nil {
		return cfg, err
	}

	err = LoadTLSConfig(cfg, store)
	if err != nil {
		return cfg, err
	}

	return cfg, err
}
