package coraza

import (
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/ingress/pkg/converter"
	"github.com/corazawaf/coraza-caddy"
	"strings"
)

type Plugin struct{}

func (p Plugin) IngressPlugin() converter.PluginInfo {
	return converter.PluginInfo{
		Name:     "ingress.coraza",
		Priority: 50,
		New:      func() converter.Plugin { return new(Plugin) },
	}
}

func (p Plugin) IngressHandler(input converter.IngressMiddlewareInput) (*caddyhttp.Route, error) {
	ing := input.Ingress

	if getAnnotation(ing, enableWAF) == "true" {
		directives := getAnnotation(ing, modsecurityDirectives)
		includes := getAnnotation(ing, modsecurityIncludes)

		finalIncludes := []string{"/etc/caddy-ingress-controller/coraza.conf-recommended"}
		if getAnnotation(ing, enableOWASPCoreRules) == "true" {
			finalIncludes = append(
				finalIncludes,
				"/etc/caddy-ingress-controller/coreruleset/crs-setup.conf",
				"/etc/caddy-ingress-controller/coreruleset/rules/*.conf",
			)
		}
		for _, file := range strings.Split(includes, ",") {
			finalIncludes = append(finalIncludes, file)
		}

		handler := caddyconfig.JSONModuleObject(
			coraza.Middleware{Include: finalIncludes, Directives: directives},
			"handler", "waf", nil,
		)

		input.Route.HandlersRaw = append(input.Route.HandlersRaw, handler)
	}

	return input.Route, nil
}

func init() {
	converter.RegisterPlugin(Plugin{})
}

// Interface guards
var (
	_ = converter.IngressMiddleware(Plugin{})
)
