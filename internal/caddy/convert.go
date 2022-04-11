package caddy

import (
	"github.com/caddyserver/ingress/pkg/converter"
	"github.com/caddyserver/ingress/pkg/store"

	// Load default plugins
	_ "github.com/caddyserver/ingress/internal/caddy/global"
	_ "github.com/caddyserver/ingress/internal/caddy/ingress"
)

type Converter struct{}

func (c Converter) ConvertToCaddyConfig(store *store.Store) (interface{}, error) {
	cfg := converter.NewConfig()

	for _, p := range converter.Plugins(store.Options.PluginsOrder) {
		if m, ok := p.(converter.GlobalMiddleware); ok {
			err := m.GlobalHandler(cfg, store)
			if err != nil {
				return cfg, err
			}
		}
	}
	return cfg, nil
}
