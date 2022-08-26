package ingress

import (
	"fmt"
	"strings"

	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
	"github.com/caddyserver/ingress/pkg/converter"
)

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

	// TODO :-
	// when setting the upstream url we should bypass kube-dns and get the ip address of
	// the pod for the deployment we are proxying to so that we can proxy to that ip address port.
	// this is good for session affinity and increases performance.
	clusterHostName := fmt.Sprintf("%v.%v.svc.cluster.local:%d", path.Backend.Service.Name, ing.Namespace, path.Backend.Service.Port.Number)

	transport := &reverseproxy.HTTPTransport{}

	if backendProtocol == "https" {
		transport.TLS = &reverseproxy.TLSConfig{
			InsecureSkipVerify: getAnnotationBool(ing, insecureSkipVerify, true),
		}
	}

	handler := reverseproxy.Handler{
		TransportRaw: caddyconfig.JSONModuleObject(transport, "protocol", "http", nil),
		Upstreams: reverseproxy.UpstreamPool{
			{Dial: clusterHostName},
		},
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

func init() {
	converter.RegisterPlugin(ReverseProxyPlugin{})
}

// Interface guards
var (
	_ = converter.IngressMiddleware(ReverseProxyPlugin{})
)
