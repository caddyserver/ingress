package global

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/ingress/pkg/converter"
	"github.com/caddyserver/ingress/pkg/store"
)

type HealthzPlugin struct{}

func (p HealthzPlugin) IngressPlugin() converter.PluginInfo {
	return converter.PluginInfo{
		Name:     "healthz",
		Priority: -20,
		New:      func() converter.Plugin { return new(HealthzPlugin) },
	}
}

func init() {
	converter.RegisterPlugin(HealthzPlugin{})
}

func (p HealthzPlugin) GlobalHandler(config *converter.Config, store *store.Store) error {
	healthzHandler := caddyhttp.StaticResponse{StatusCode: caddyhttp.WeakString(strconv.Itoa(http.StatusOK))}

	healthzRoute := caddyhttp.Route{
		HandlersRaw: []json.RawMessage{
			caddyconfig.JSONModuleObject(healthzHandler, "handler", healthzHandler.CaddyModule().ID.Name(), nil),
		},
		MatcherSetsRaw: []caddy.ModuleMap{{
			"path": caddyconfig.JSON(caddyhttp.MatchPath{"/healthz"}, nil),
		}},
	}

	config.GetMetricsServer().Routes = append(config.GetMetricsServer().Routes, healthzRoute)
	return nil
}

// Interface guards
var (
	_ = converter.GlobalMiddleware(HealthzPlugin{})
)
