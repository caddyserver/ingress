package caddy

import (
	"encoding/json"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddytls"
	"github.com/caddyserver/ingress/pkg/controller"
)


// LoadTLSConfig configure caddy when some ingresses have TLS certs
func LoadTLSConfig(config *Config, store *controller.Store) error {
	tlsApp := config.Apps["tls"].(*caddytls.TLS)
	httpApp := config.Apps["http"].(*caddyhttp.App)

	var hosts []string

	// Get all Hosts subject to custom TLS certs
	for _, ing := range store.Ingresses {
		for _, tlsRule := range ing.Spec.TLS {
			for _, h := range tlsRule.Hosts {
				hosts = append(hosts, h)
			}
		}
	}

	if len(hosts) > 0 {
		tlsApp.CertificatesRaw["load_folders"] = json.RawMessage(`["` + controller.CertFolder + `"]`)
		// do not manage certificates for those hosts
		httpApp.Servers[HttpServer].AutoHTTPS.SkipCerts = hosts
	}
	return nil
}
