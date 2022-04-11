package global

import (
	"github.com/caddyserver/ingress/pkg/converter"
	"github.com/caddyserver/ingress/pkg/store"
)

type SecretsStorePlugin struct{}

func (p SecretsStorePlugin) IngressPlugin() converter.PluginInfo {
	return converter.PluginInfo{
		Name: "secrets_store",
		New:  func() converter.Plugin { return new(SecretsStorePlugin) },
	}
}

func init() {
	converter.RegisterPlugin(SecretsStorePlugin{})
}

func (p SecretsStorePlugin) GlobalHandler(config *converter.Config, store *store.Store) error {
	config.Storage = converter.Storage{
		System: "secret_store",
		StorageValues: converter.StorageValues{
			Namespace: store.CurrentPod.Namespace,
			LeaseId:   store.Options.LeaseId,
		},
	}

	return nil
}

// Interface guards
var (
	_ = converter.GlobalMiddleware(SecretsStorePlugin{})
)
