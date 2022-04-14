package ingress

import (
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/rewrite"
	"github.com/caddyserver/ingress/pkg/converter"
)

type RewritePlugin struct{}

func (p RewritePlugin) IngressPlugin() converter.PluginInfo {
	return converter.PluginInfo{
		Name:     "ingress.rewrite",
		Priority: 10,
		New:      func() converter.Plugin { return new(RewritePlugin) },
	}
}

// IngressHandler Converts rewrite annotations to rewrite handler
func (p RewritePlugin) IngressHandler(input converter.IngressMiddlewareInput) (*caddyhttp.Route, error) {
	ing := input.Ingress

	rewriteTo := getAnnotation(ing, rewriteToAnnotation)
	if rewriteTo != "" {
		handler := caddyconfig.JSONModuleObject(
			rewrite.Rewrite{URI: rewriteTo},
			"handler", "rewrite", nil,
		)

		input.Route.HandlersRaw = append(input.Route.HandlersRaw, handler)
	}

	rewriteStripPrefix := getAnnotation(ing, rewriteStripPrefixAnnotation)
	if rewriteStripPrefix != "" {
		handler := caddyconfig.JSONModuleObject(
			rewrite.Rewrite{StripPathPrefix: rewriteStripPrefix},
			"handler", "rewrite", nil,
		)

		input.Route.HandlersRaw = append(input.Route.HandlersRaw, handler)
	}
	return input.Route, nil
}

func init() {
	converter.RegisterPlugin(RewritePlugin{})
}

// Interface guards
var (
	_ = converter.IngressMiddleware(RewritePlugin{})
)
