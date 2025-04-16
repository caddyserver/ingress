package global

import (
	"encoding/json"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/ingress/pkg/converter"
	"github.com/caddyserver/ingress/pkg/store"
)

type IngressPlugin struct{}

func (p IngressPlugin) IngressPlugin() converter.PluginInfo {
	return converter.PluginInfo{
		Name: "ingress",
		New:  func() converter.Plugin { return new(IngressPlugin) },
	}
}

func init() {
	converter.RegisterPlugin(IngressPlugin{})
}

func (p IngressPlugin) GlobalHandler(config *converter.Config, store *store.Store) error {
	ingressHandlers := make([]converter.IngressMiddleware, 0)
	for _, plugin := range converter.Plugins(store.Options.PluginsOrder) {
		if m, ok := plugin.(converter.IngressMiddleware); ok {
			ingressHandlers = append(ingressHandlers, m)
		}
	}

	// create a server route for each ingress route
	var routes caddyhttp.RouteList
	for _, ing := range store.Ingresses() {
		for _, rule := range ing.Spec.Rules {
			for _, path := range rule.HTTP.Paths {
				r := &caddyhttp.Route{
					HandlersRaw:    []json.RawMessage{},
					MatcherSetsRaw: []caddy.ModuleMap{},
				}

				for _, middleware := range ingressHandlers {
					newRoute, err := middleware.IngressHandler(converter.IngressMiddlewareInput{
						Config:  config,
						Store:   store,
						Ingress: ing,
						Rule:    rule,
						Path:    path,
						Route:   r,
					})
					if err != nil {
						return err
					}
					r = newRoute
				}

				routes = append(routes, *r)
			}
		}
	}

	config.GetHTTPServer().Routes = routes
	return nil
}

// Interface guards
var (
	_ = converter.GlobalMiddleware(IngressPlugin{})
)
