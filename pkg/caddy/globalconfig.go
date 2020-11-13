package caddy

import (
	caddy2 "github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddytls"
	"github.com/caddyserver/ingress/pkg/controller"
)

// LoadConfigMapOptions load options from ConfigMap
func LoadConfigMapOptions(config *Config, store *controller.Store) error {
	cfgMap := store.ConfigMap

	tlsApp := config.Apps["tls"].(*caddytls.TLS)

	if cfgMap.Debug {
		config.Logging.Logs = map[string]*caddy2.CustomLog{"default": {Level: "DEBUG"}}
	}

	if cfgMap.AcmeCA != "" || cfgMap.Email != "" {
		acmeIssuer := caddytls.ACMEIssuer{}

		if cfgMap.AcmeCA != "" {
			acmeIssuer.CA = cfgMap.AcmeCA
		}

		if cfgMap.Email != "" {
			acmeIssuer.Email = cfgMap.Email
		}

		tlsApp.Automation = &caddytls.AutomationConfig{
			Policies: []*caddytls.AutomationPolicy{
				{IssuerRaw: caddyconfig.JSONModuleObject(acmeIssuer, "module", "acme", nil)},
			},
		}
	}

	return nil
}
