package global

import (
	"encoding/json"
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
	"github.com/caddyserver/caddy/v2/modules/caddytls"
	"github.com/caddyserver/ingress/internal/caddy/ingress"
	"github.com/caddyserver/ingress/pkg/converter"
	"github.com/caddyserver/ingress/pkg/store"
	v1 "k8s.io/api/networking/v1"
	"strings"
)

type DefaultBackendPlugin struct{}

func (p DefaultBackendPlugin) IngressPlugin() converter.PluginInfo {
	return converter.PluginInfo{
		Name: "default-backend",
		// Should go after everything including the ingress sort plugin.
		Priority: -50,
		New:      func() converter.Plugin { return new(DefaultBackendPlugin) },
	}
}

func init() {
	converter.RegisterPlugin(DefaultBackendPlugin{})
}

// GlobalHandler This plugin define a default backend and also define onDemandTLS config
// if configured by the ingress (through annotations).
func (p DefaultBackendPlugin) GlobalHandler(config *converter.Config, store *store.Store) error {
	// Find default backend ingress
	// Use onDemandTLS annotations, fallback on configmap
	var defaultIng *v1.Ingress
	for _, ing := range store.Ingresses {
		if ing.Spec.DefaultBackend != nil {
			defaultIng = ing
			break
		}
	}
	if defaultIng == nil {
		return nil
	}

	clusterHostName := fmt.Sprintf(
		"%v.%v.svc.cluster.local:%d",
		defaultIng.Spec.DefaultBackend.Service.Name,
		defaultIng.Namespace,
		defaultIng.Spec.DefaultBackend.Service.Port.Number,
	)
	backendProtocol := strings.ToLower(ingress.GetAnnotation(defaultIng, ingress.BackendProtocol))

	transport := &reverseproxy.HTTPTransport{}
	if backendProtocol == "https" {
		transport.TLS = &reverseproxy.TLSConfig{
			InsecureSkipVerify: ingress.GetAnnotationBool(defaultIng, ingress.InsecureSkipVerify, true),
		}
	}

	defaultRoute := caddyhttp.Route{
		HandlersRaw: []json.RawMessage{caddyconfig.JSONModuleObject(
			reverseproxy.Handler{
				TransportRaw: caddyconfig.JSONModuleObject(transport, "protocol", "http", nil),
				Upstreams:    reverseproxy.UpstreamPool{{Dial: clusterHostName}}},
			"handler",
			"reverse_proxy",
			nil,
		)},
	}
	config.GetHTTPServer().Routes = append(config.GetHTTPServer().Routes, defaultRoute)

	// If TLS Automation is enabled, allow overriding on demand config
	tlsApp := config.GetTLSApp()
	if tlsApp.Automation != nil {
		overridden := false
		onDemandConfig := tlsApp.Automation.OnDemand
		if onDemandConfig == nil {
			onDemandConfig = &caddytls.OnDemandConfig{
				RateLimit: &caddytls.RateLimit{},
			}
		}

		ask := ingress.GetAnnotation(defaultIng, ingress.OnDemandTLSAsk)
		if ask != "" {
			overridden = true
			onDemandConfig.Ask = ask
		}

		rateLimitInterval := ingress.GetAnnotation(defaultIng, ingress.OnDemandTLSRateLimitInterval)
		if rateLimitInterval != "" {
			dur, err := caddy.ParseDuration(rateLimitInterval)
			if err == nil {
				onDemandConfig.RateLimit.Interval = caddy.Duration(dur)
				overridden = true
			}
		}

		rateLimitBurst := ingress.GetAnnotationInt(defaultIng, ingress.OnDemandTLSRateLimitBurst, 0)
		if rateLimitBurst != 0 {
			overridden = true
			onDemandConfig.RateLimit.Burst = rateLimitBurst
		}

		if ingress.HasAnnotation(defaultIng, ingress.OnDemandTLS) {
			tlsPolicy := tlsApp.Automation.Policies[0]
			onDemandTLSEnabled := ingress.GetAnnotationBool(defaultIng, ingress.OnDemandTLS, tlsPolicy.OnDemand)
			tlsPolicy.OnDemand = onDemandTLSEnabled
		}

		if overridden {
			tlsApp.Automation.OnDemand = onDemandConfig
		}
	}

	return nil
}

// Interface guards
var (
	_ = converter.GlobalMiddleware(DefaultBackendPlugin{})
)
