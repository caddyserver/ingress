package ingress

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/ingress/pkg/converter"
)

type RedirectPlugin struct{}

func (p RedirectPlugin) IngressPlugin() converter.PluginInfo {
	return converter.PluginInfo{
		Name:     "ingress.redirect",
		Priority: 10,
		New:      func() converter.Plugin { return new(RedirectPlugin) },
	}
}

// IngressHandler Converts redirect annotations to static_response handler
func (p RedirectPlugin) IngressHandler(input converter.IngressMiddlewareInput) (*caddyhttp.Route, error) {
	ing := input.Ingress

	var code string = "301"

	// Logic taken from Caddy's parseRedir builtin function
	redirectCode := getAnnotation(ing, permanentRedirectCodeAnnotation)
	if redirectCode != "" {
		switch redirectCode {
		case "permanent":
			code = "301"
		case "temporary":
			code = "302"
		default:
			codeInt, err := strconv.Atoi(redirectCode)
			if err != nil {
				return nil, fmt.Errorf("not a supported redir code type or not valid integer: '%s'", redirectCode)
			}

			if codeInt < 300 || (codeInt > 399 && codeInt != 401) {
				return nil, fmt.Errorf("redir code not in the 3xx range or 401: '%v'", codeInt)
			}

			code = redirectCode
		}
	}

	redirectTo := getAnnotation(ing, permanentRedirectAnnotation)
	if redirectTo != "" {
		handler := caddyconfig.JSONModuleObject(
			caddyhttp.StaticResponse{
				StatusCode: caddyhttp.WeakString(code),
				Headers:    http.Header{"Location": []string{redirectTo}},
			},
			"handler", "static_response", nil,
		)

		input.Route.HandlersRaw = append(input.Route.HandlersRaw, handler)
	}

	return input.Route, nil
}

func init() {
	converter.RegisterPlugin(RedirectPlugin{})
}

// Interface guards
var (
	_ = converter.IngressMiddleware(RedirectPlugin{})
)
