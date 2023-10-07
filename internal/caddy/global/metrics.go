package global

import (
	"encoding/json"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/ingress/pkg/converter"
	"github.com/caddyserver/ingress/pkg/store"
)

type MetricsPlugin struct{}

func (p MetricsPlugin) IngressPlugin() converter.PluginInfo {
	return converter.PluginInfo{
		Name: "metrics",
		New:  func() converter.Plugin { return new(MetricsPlugin) },
	}
}

func init() {
	converter.RegisterPlugin(MetricsPlugin{})
}

func (p MetricsPlugin) GlobalHandler(config *converter.Config, store *store.Store) error {
	if store.ConfigMap.Metrics {
		metricsRoute := caddyhttp.Route{
			HandlersRaw: []json.RawMessage{json.RawMessage(`{ "handler": "metrics" }`)},
			MatcherSetsRaw: []caddy.ModuleMap{{
				"path": caddyconfig.JSON(caddyhttp.MatchPath{"/metrics"}, nil),
			}},
		}

		config.GetMetricsServer().Routes = append(config.GetMetricsServer().Routes, metricsRoute)
	}
	return nil
}

// Interface guards
var (
	_ = converter.GlobalMiddleware(MetricsPlugin{})
)
