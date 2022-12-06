package converter

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddytls"
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

func (c Config) GetHTTPServer() *caddyhttp.Server {
	return c.Apps["http"].(*caddyhttp.App).Servers[HttpServer]
}

func (c Config) GetMetricsServer() *caddyhttp.Server {
	return c.Apps["http"].(*caddyhttp.App).Servers[MetricsServer]
}

func (c Config) GetTLSApp() *caddytls.TLS {
	return c.Apps["tls"].(*caddytls.TLS)
}

func NewConfig() *Config {
	return &Config{
		Logging: caddy.Logging{},
		Apps: map[string]interface{}{
			"tls": &caddytls.TLS{CertificatesRaw: caddy.ModuleMap{}},
			"http": &caddyhttp.App{
				Servers: map[string]*caddyhttp.Server{
					HttpServer: {
						AutoHTTPS: &caddyhttp.AutoHTTPSConfig{},
						// Listen to both :80 and :443 ports in order
						// to use the same listener wrappers (PROXY protocol use it)
						Listen: []string{":80", ":443"},
						TLSConnPolicies: caddytls.ConnectionPolicies{
							&caddytls.ConnectionPolicy{},
						},
					},
					MetricsServer: {
						Listen:    []string{":9765"},
						AutoHTTPS: &caddyhttp.AutoHTTPSConfig{Disabled: true},
					},
				},
			},
		},
	}
}
