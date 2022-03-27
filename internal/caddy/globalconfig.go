package caddy

import (
	"encoding/json"
	caddy2 "github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddytls"
	"github.com/caddyserver/ingress/internal/controller"
)

// LoadConfigMapOptions load options from ConfigMap
func LoadConfigMapOptions(config *Config, store *controller.Store) error {
	cfgMap := store.ConfigMap

	tlsApp := config.Apps["tls"].(*caddytls.TLS)
	httpServer := config.Apps["http"].(*caddyhttp.App).Servers[HttpServer]

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

		var onDemandConfig *caddytls.OnDemandConfig
		if cfgMap.OnDemandTLS {
			onDemandConfig = &caddytls.OnDemandConfig{
				RateLimit: &caddytls.RateLimit{
					Interval: cfgMap.OnDemandRateLimitInterval,
					Burst:    cfgMap.OnDemandRateLimitBurst,
				},
				Ask: cfgMap.OnDemandAsk,
			}
		}

		tlsApp.Automation = &caddytls.AutomationConfig{
			OnDemand:          onDemandConfig,
			OCSPCheckInterval: cfgMap.OCSPCheckInterval,
			Policies: []*caddytls.AutomationPolicy{
				{
					IssuersRaw: []json.RawMessage{
						caddyconfig.JSONModuleObject(acmeIssuer, "module", "acme", nil),
					},
					OnDemand: cfgMap.OnDemandTLS,
				},
			},
		}
	}

	if cfgMap.ProxyProtocol {
		httpServer.ListenerWrappersRaw = []json.RawMessage{
			json.RawMessage(`{"wrapper":"proxy_protocol"}`),
			json.RawMessage(`{"wrapper":"tls"}`),
		}
	}
	return nil
}
