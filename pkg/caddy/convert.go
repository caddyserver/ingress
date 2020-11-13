package caddy

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddytls"
	"github.com/caddyserver/ingress/pkg/controller"
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
	Admin   caddy.AdminConfig      `json:"admin,omitempty"`
	Storage Storage                `json:"storage"`
	Apps    map[string]interface{} `json:"apps"`
	Logging caddy.Logging          `json:"logging"`
}

type Converter struct{}

const (
	HttpServer = "ingress_server"
)

func newConfig() (*Config, error) {
	cfg := &Config{
		Logging: caddy.Logging{},
		Apps: map[string]interface{}{
			"tls": &caddytls.TLS{
				CertificatesRaw: caddy.ModuleMap{},
			},
			"http": &caddyhttp.App{
				Servers: map[string]*caddyhttp.Server{
					HttpServer: {
						AutoHTTPS: &caddyhttp.AutoHTTPSConfig{},
						Listen:    []string{":443"},
					},
				},
			},
		},
		Storage: Storage{
			System:        "secret_store",
			StorageValues: StorageValues{},
		},
	}

	return cfg, nil
}

func (c Converter) ConvertToCaddyConfig(store *controller.Store) (interface{}, error) {
	cfg, err := newConfig()

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
