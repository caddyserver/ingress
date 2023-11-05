package global

import (
	"encoding/json"
	"slices"

	"github.com/caddyserver/ingress/internal/controller"
	"github.com/caddyserver/ingress/pkg/converter"
	"github.com/caddyserver/ingress/pkg/store"
)

type TLSPlugin struct{}

func (p TLSPlugin) IngressPlugin() converter.PluginInfo {
	return converter.PluginInfo{
		Name: "tls",
		New:  func() converter.Plugin { return new(TLSPlugin) },
	}
}

func init() {
	converter.RegisterPlugin(TLSPlugin{})
}

func (p TLSPlugin) GlobalHandler(config *converter.Config, store *store.Store) error {
	tlsApp := config.GetTLSApp()
	httpServer := config.GetHTTPServer()

	var hosts []string

	// Get all Hosts subject to custom TLS certs
	for _, ing := range store.Ingresses {
		for _, tlsRule := range ing.Spec.TLS {
			for _, h := range tlsRule.Hosts {
				if !slices.Contains(hosts, h) {
					hosts = append(hosts, h)
				}
			}
		}
	}

	if len(hosts) > 0 {
		tlsApp.CertificatesRaw["load_folders"] = json.RawMessage(`["` + controller.CertFolder + `"]`)
		// do not manage certificates for those hosts
		httpServer.AutoHTTPS.SkipCerts = hosts
	}
	return nil
}

// Interface guards
var (
	_ = converter.GlobalMiddleware(TLSPlugin{})
)
