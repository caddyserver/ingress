package ingress

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
	"github.com/caddyserver/ingress/internal/controller"
	"github.com/caddyserver/ingress/pkg/converter"
)

type ReverseProxyPlugin struct {
	diags controller.Diagnostics
}

func (p ReverseProxyPlugin) IngressPlugin() converter.PluginInfo {
	return converter.PluginInfo{
		Name: "ingress.reverseproxy",
		// Should always go last by default
		Priority: -10,
		New: func() converter.Plugin {
			return &ReverseProxyPlugin{
				diags: make(controller.Diagnostics),
			}
		},
	}
}

// IngressHandler Add a reverse proxy handler to the route
func (p ReverseProxyPlugin) IngressHandler(input converter.IngressMiddlewareInput) (*caddyhttp.Route, error) {
	logger := input.Store.Logger
	path := input.Path
	ing := input.Ingress
	backendProtocol := strings.ToLower(getAnnotation(ing, backendProtocol))
	trustedProxiesAnnotation := strings.ToLower(getAnnotation(ing, trustedProxies))

	serviceRef := path.Backend.Service
	if serviceRef == nil {
		p.diags.Warnf(logger, "Ingress %s/%s uses a non-service backend, which is not supported, and will be ignored", ing.Namespace, ing.Name)
		return input.Route, nil
	}

	serviceName := fmt.Sprintf("%s/%s", ing.Namespace, serviceRef.Name)
	service := input.Store.Service(serviceName)
	if service == nil {
		p.diags.Warnf(logger, "Ingress %s/%s references unknown service %s and will be ignored", ing.Namespace, ing.Name, serviceRef.Name)
		return input.Route, nil
	}

	var upstreams reverseproxy.UpstreamPool
	if service.Spec.Type == "ExternalName" {
		// Create a single upstream for type=ExternalName.
		if serviceRef.Port.Number == 0 {
			p.diags.Warnf(logger, "Ingress %s/%s references service %s with type=ExternalName and a named port, which is not supported, and will be ignored", ing.Namespace, ing.Name, service.Name)
			return input.Route, nil
		}
		upstreams = reverseproxy.UpstreamPool{
			{Dial: formatDialAddr(service.Spec.ExternalName, serviceRef.Port.Number)},
		}
	} else {
		// Find the TargetPort on the Service.
		var targetPort int32
		for _, port := range service.Spec.Ports {
			if (serviceRef.Port.Number != 0 && port.Port == serviceRef.Port.Number) ||
				(serviceRef.Port.Name != "" && port.Name == serviceRef.Port.Name) {
				targetPort = int32(port.TargetPort.IntValue())
				if targetPort == 0 {
					p.diags.Warnf(logger, "Ingress %s/%s references service %s with a named target port, which is not supported, and will be ignored", ing.Namespace, ing.Name, service.Name)
					return input.Route, nil
				}
				break
			}
		}
		if targetPort == 0 {
			p.diags.Warnf(logger, "Ingress %s/%s references an unknown port on service %s, and will be ignored", ing.Namespace, ing.Name, service.Name)
			return input.Route, nil
		}

		// Create upstreams for each endpoint.
		for _, es := range input.Store.EndpointSlicesByService(serviceName) {
			for _, e := range es.Endpoints {
				if e.Conditions.Ready == nil || *e.Conditions.Ready {
					for _, addr := range e.Addresses {
						upstreams = append(upstreams, &reverseproxy.Upstream{
							Dial: formatDialAddr(addr, targetPort),
						})
					}
				}
			}
		}
	}

	transport := &reverseproxy.HTTPTransport{}
	if backendProtocol == "https" {
		transport.TLS = &reverseproxy.TLSConfig{
			InsecureSkipVerify: getAnnotationBool(ing, insecureSkipVerify, true),
		}
	}

	var err error
	var parsedProxies []string
	if trustedProxiesAnnotation != "" {
		trustedProxies := strings.Split(trustedProxiesAnnotation, ",")
		parsedProxies, err = parseTrustedProxies(trustedProxies)
		if err != nil {
			return nil, err
		}
	}

	handler := reverseproxy.Handler{
		TransportRaw:   caddyconfig.JSONModuleObject(transport, "protocol", "http", nil),
		Upstreams:      upstreams,
		TrustedProxies: parsedProxies,
	}

	handlerModule := caddyconfig.JSONModuleObject(
		handler,
		"handler",
		"reverse_proxy",
		nil,
	)
	input.Route.HandlersRaw = append(input.Route.HandlersRaw, handlerModule)
	return input.Route, nil
}

func (p ReverseProxyPlugin) Finalize() {
	p.diags.Gc()
}

func formatDialAddr(host string, port int32) string {
	if strings.Contains(host, ":") {
		return fmt.Sprintf("[%s]:%d", host, port)
	}
	return fmt.Sprintf("%s:%d", host, port)
}

// Copied from https://github.com/caddyserver/caddy/blob/21af88fefc9a8239a024f635f1c6fdd9defd7eb7/modules/caddyhttp/reverseproxy/reverseproxy.go#L270-L286
func parseTrustedProxies(trustedProxies []string) (parsedProxies []string, err error) {
	for _, trustedProxy := range trustedProxies {
		trustedProxy = strings.TrimSpace(trustedProxy)
		if strings.Contains(trustedProxy, "/") {
			ipNet, err := netip.ParsePrefix(trustedProxy)
			if err != nil {
				return nil, fmt.Errorf("failed to parse IP: %q", trustedProxy)
			}
			parsedProxies = append(parsedProxies, ipNet.String())
		} else {
			ipAddr, err := netip.ParseAddr(trustedProxy)
			if err != nil {
				return nil, fmt.Errorf("failed to parse IP: %q", trustedProxy)
			}
			ipNew := netip.PrefixFrom(ipAddr, ipAddr.BitLen())
			parsedProxies = append(parsedProxies, ipNew.String())
		}
	}
	return parsedProxies, nil
}

func init() {
	converter.RegisterPlugin(ReverseProxyPlugin{})
}

// Interface guards
var (
	_ = converter.IngressMiddleware(ReverseProxyPlugin{})
	_ = converter.Finalizer(ReverseProxyPlugin{})
)
