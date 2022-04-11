package converter

import (
	"fmt"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/ingress/pkg/store"
	v1 "k8s.io/api/networking/v1"
	"sort"
)

const (
	HttpServer    = "ingress_server"
	MetricsServer = "metrics_server"
)

// GlobalMiddleware is called with a default caddy config
// already configured with:
//	- Secret storage store
//	- A TLS App (https://caddyserver.com/docs/json/apps/tls/)
//	- A HTTP App with an HTTP server listening to 80 443 ports (https://caddyserver.com/docs/json/apps/http/)
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

type Plugin interface {
	IngressPlugin() PluginInfo
}

type PluginInfo struct {
	Name string
	New  func() Plugin
}

func RegisterPlugin(m Plugin) {
	plugin := m.IngressPlugin()

	if _, ok := plugins[plugin.Name]; ok {
		panic(fmt.Sprintf("plugin already registered: %s", plugin.Name))
	}
	plugins[plugin.Name] = plugin
}

func getOrderIndex(order []string, plugin string) int {
	for idx, o := range order {
		if plugin == o {
			return idx
		}
	}
	return -1
}

// Plugins return a sorted array of plugin instances.
// Sort is taken from the `order` parameter. Plugin names specified in the parameter
// will be taken first, then other plugins will be added in alphabetical order.
func Plugins(order []string) []Plugin {
	sortedPlugins := make([]PluginInfo, 0, len(plugins))
	for _, p := range plugins {
		sortedPlugins = append(sortedPlugins, p)
	}

	sort.Slice(sortedPlugins, func(i, j int) bool {
		iSortedIdx := getOrderIndex(order, sortedPlugins[i].Name)
		jSortedIdx := getOrderIndex(order, sortedPlugins[j].Name)

		if iSortedIdx == -1 && jSortedIdx == -1 {
			return sortedPlugins[i].Name < sortedPlugins[j].Name
		}
		return iSortedIdx < jSortedIdx
	})

	pluginArr := make([]Plugin, 0, len(plugins))
	for _, p := range sortedPlugins {
		pluginArr = append(pluginArr, p.New())
	}
	return pluginArr
}

var (
	plugins = make(map[string]PluginInfo)
)
