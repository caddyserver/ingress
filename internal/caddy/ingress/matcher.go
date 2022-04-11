package ingress

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/ingress/pkg/converter"
	v1 "k8s.io/api/networking/v1"
)

type MatcherPlugin struct{}

func (p MatcherPlugin) IngressPlugin() converter.PluginInfo {
	return converter.PluginInfo{
		Name: "ingress.matcher",
		New:  func() converter.Plugin { return new(MatcherPlugin) },
	}
}

// IngressHandler Generate matchers for the route.
func (p MatcherPlugin) IngressHandler(input converter.IngressMiddlewareInput) (*caddyhttp.Route, error) {
	match := caddy.ModuleMap{}

	if getAnnotation(input.Ingress, disableSSLRedirect) != "true" {
		match["protocol"] = caddyconfig.JSON(caddyhttp.MatchProtocol("https"), nil)
	}

	if input.Rule.Host != "" {
		match["host"] = caddyconfig.JSON(caddyhttp.MatchHost{input.Rule.Host}, nil)
	}

	if input.Path.Path != "" {
		p := input.Path.Path

		if *input.Path.PathType == v1.PathTypePrefix {
			p += "*"
		}
		match["path"] = caddyconfig.JSON(caddyhttp.MatchPath{p}, nil)
	}

	input.Route.MatcherSetsRaw = append(input.Route.MatcherSetsRaw, match)
	return input.Route, nil
}

func init() {
	converter.RegisterPlugin(MatcherPlugin{})
}

// Interface guards
var (
	_ = converter.IngressMiddleware(MatcherPlugin{})
)
