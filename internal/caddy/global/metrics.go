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
	httpApp := config.Apps["http"].(*caddyhttp.App)

	if store.ConfigMap.Metrics {
		httpApp.Servers[converter.MetricsServer] = &caddyhttp.Server{
			Listen:    []string{":9765"},
			AutoHTTPS: &caddyhttp.AutoHTTPSConfig{Disabled: true},
			Routes: []caddyhttp.Route{{
				HandlersRaw: []json.RawMessage{json.RawMessage(`{ "handler": "metrics" }`)},
				MatcherSetsRaw: []caddy.ModuleMap{{
					"path": caddyconfig.JSON(caddyhttp.MatchPath{"/metrics"}, nil),
				}},
			}},
		}

	}
	return nil
}

// Interface guards
var (
	_ = converter.GlobalMiddleware(MetricsPlugin{})
)
