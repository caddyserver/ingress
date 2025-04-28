package ingress

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
	"github.com/caddyserver/ingress/pkg/converter"
)

var ClusterDomain = "cluster.local"

type ReverseProxyPlugin struct{}

func (p ReverseProxyPlugin) IngressPlugin() converter.PluginInfo {
	return converter.PluginInfo{
		Name: "ingress.reverseproxy",
		// Should always go last by default
		Priority: -10,
		New:      func() converter.Plugin { return new(ReverseProxyPlugin) },
	}
}

// IngressHandler Add a reverse proxy handler to the route
func (p ReverseProxyPlugin) IngressHandler(input converter.IngressMiddlewareInput) (*caddyhttp.Route, error) {
	path := input.Path
	ing := input.Ingress
	backendProtocol := strings.ToLower(getAnnotation(ing, backendProtocol))
	trustedProxiesAnnotation := strings.ToLower(getAnnotation(ing, trustedProxies))

	// TODO :-
	// when setting the upstream url we should bypass kube-dns and get the ip address of
	// the pod for the deployment we are proxying to so that we can proxy to that ip address port.
	// this is good for session affinity and increases performance.
	clusterHostName := fmt.Sprintf("%v.%v.svc.%s:%d", path.Backend.Service.Name, ing.Namespace, ClusterDomain, path.Backend.Service.Port.Number)

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
		TransportRaw: caddyconfig.JSONModuleObject(transport, "protocol", "http", nil),
		Upstreams: reverseproxy.UpstreamPool{
			{Dial: clusterHostName},
		},
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
)
