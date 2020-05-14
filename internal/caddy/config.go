package caddy

import (
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
}

// ControllerConfig represents ingress controller config received through cli arguments.
type ControllerConfig struct {
	Email              string
	AutomaticTLS       bool
	AutomaticRedirects bool
	TLSUseStaging      bool
	WatchNamespace     string
}

// NewConfig returns a plain slate caddy2 config file.
func NewConfig(namespace string, cfg ControllerConfig) *Config {
	acmeIssuer := caddytls.ACMEIssuer{
		CA:    getCAEndpoint(cfg.TLSUseStaging),
		Email: cfg.Email}

	listenPorts := []string{":443"}

	if !cfg.AutomaticRedirects {
		listenPorts = append(listenPorts, ":80")
	}

	return &Config{
		Storage: Storage{
			System: "secret_store",
			StorageValues: StorageValues{
				Namespace: namespace,
			},
		},
		Apps: map[string]interface{}{
			"tls": caddytls.TLS{
				Automation: &caddytls.AutomationConfig{
					Policies: []*caddytls.AutomationPolicy{
						{
							IssuerRaw: caddyconfig.JSONModuleObject(acmeIssuer, "module", "acme", nil),
						},
					},
				},
				CertificatesRaw: caddy.ModuleMap{},
			},
			"http": caddyhttp.App{
				Servers: map[string]*caddyhttp.Server{
					"ingress_server": {
						AutoHTTPS: &caddyhttp.AutoHTTPSConfig{
							Disabled: !cfg.AutomaticTLS,
							Skip:     make([]string, 0),
						},
						Listen: listenPorts,
					},
				},
			},
		},
	}
}

func getCAEndpoint(useStaging bool) string {
	if useStaging {
		return "https://acme-staging-v02.api.letsencrypt.org/directory"
	}
	return ""
}
