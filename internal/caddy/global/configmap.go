package global

import (
	"encoding/json"

	caddy2 "github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddytls"
	"github.com/caddyserver/ingress/pkg/converter"
	"github.com/caddyserver/ingress/pkg/store"
	"github.com/mholt/acmez/acme"
)

type ConfigMapPlugin struct{}

func init() {
	converter.RegisterPlugin(ConfigMapPlugin{})
}

func (p ConfigMapPlugin) IngressPlugin() converter.PluginInfo {
	return converter.PluginInfo{
		Name: "configmap",
		New:  func() converter.Plugin { return new(ConfigMapPlugin) },
	}
}

func (p ConfigMapPlugin) GlobalHandler(config *converter.Config, store *store.Store) error {
	cfgMap := store.ConfigMap

	tlsApp := config.GetTLSApp()
	httpServer := config.GetHTTPServer()

	if cfgMap.Debug {
		config.Logging.Logs = map[string]*caddy2.CustomLog{"default": {BaseLog: caddy2.BaseLog{Level: "DEBUG"}}}
	}

	if cfgMap.AcmeCA != "" || cfgMap.Email != "" {
		acmeIssuer := caddytls.ACMEIssuer{}

		if cfgMap.AcmeCA != "" {
			acmeIssuer.CA = cfgMap.AcmeCA
		}

		if cfgMap.AcmeEABKeyId != "" && cfgMap.AcmeEABMacKey != "" {
			acmeIssuer.ExternalAccount = &acme.EAB{
				KeyID:  cfgMap.AcmeEABKeyId,
				MACKey: cfgMap.AcmeEABMacKey,
			}
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

// Interface guards
var (
	_ = converter.GlobalMiddleware(ConfigMapPlugin{})
)
