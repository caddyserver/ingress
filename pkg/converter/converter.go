package converter

import (
	"fmt"
	"sort"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/ingress/pkg/store"
	v1 "k8s.io/api/networking/v1"
)

const (
	HTTPServer    = "ingress_server"
	MetricsServer = "metrics_server"
)

// GlobalMiddleware is called with a default caddy config
// already configured with:
//   - Secret storage store
//   - A TLS App (https://caddyserver.com/docs/json/apps/tls/)
//   - A HTTP App with an HTTP server listening to 80 443 ports (https://caddyserver.com/docs/json/apps/http/)
type GlobalMiddleware interface {
	GlobalHandler(config *Config, store *store.Store) error
}

type IngressMiddlewareInput struct {
	Config  *Config
	Store   *store.Store
	Ingress *v1.Ingress
	Rule    v1.IngressRule
	Path    v1.HTTPIngressPath
	Route   *caddyhttp.Route
}

// IngressMiddleware is called for each Caddy route that is generated for a specific
// ingress. It allows anyone to manipulate caddy routes before sending it to caddy.
type IngressMiddleware interface {
	IngressHandler(input IngressMiddlewareInput) (*caddyhttp.Route, error)
}

// Finalizer is an optional interface plugins can implement to perform finalization.
// Note that Finalize may be called even if an earlier plugin caused an error.
type Finalizer interface {
	Finalize()
}

type Plugin interface {
	IngressPlugin() PluginInfo
}

type PluginInfo struct {
	Name     string
	Priority int
	New      func() Plugin
}

func RegisterPlugin(m Plugin) {
	plugin := m.IngressPlugin()

	if _, ok := plugins[plugin.Name]; ok {
		panic(fmt.Sprintf("plugin already registered: %s", plugin.Name))
	}
	plugins[plugin.Name] = plugin
	pluginInstances[plugin.Name] = plugin.New()
}

func getOrderIndex(order []string, plugin string) int {
	for idx, o := range order {
		if plugin == o {
			return idx
		}
	}
	return -1
}

func sortPlugins(plugins []PluginInfo, order []string) []PluginInfo {
	sort.SliceStable(plugins, func(i, j int) bool {
		iPlugin, jPlugin := plugins[i], plugins[j]

		iSortedIdx := getOrderIndex(order, iPlugin.Name)
		jSortedIdx := getOrderIndex(order, jPlugin.Name)

		if iSortedIdx != jSortedIdx {
			return iSortedIdx > jSortedIdx
		}

		if iPlugin.Priority != jPlugin.Priority {
			return iPlugin.Priority > jPlugin.Priority
		}
		return iPlugin.Name < jPlugin.Name

	})
	return plugins
}

// Plugins return a sorted array of plugin instances.
// Sort is made following these rules:
//   - Plugins specified in the order slice will always go first (in the order specified in the slice)
//   - A Plugin with higher priority will go before a plugin with lower priority
//   - If 2 plugins have the same priority (and not in order slice), they will be sorted by plugin name
func Plugins(order []string) []Plugin {
	sortedPlugins := make([]PluginInfo, 0, len(plugins))
	for _, p := range plugins {
		sortedPlugins = append(sortedPlugins, p)
	}

	sortPlugins(sortedPlugins, order)

	pluginArr := make([]Plugin, 0, len(plugins))
	for _, p := range sortedPlugins {
		pluginArr = append(pluginArr, pluginInstances[p.Name])
	}
	return pluginArr
}

var (
	plugins         = make(map[string]PluginInfo)
	pluginInstances = make(map[string]Plugin)
)
