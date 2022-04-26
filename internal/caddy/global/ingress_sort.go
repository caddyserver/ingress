package global

import (
	"encoding/json"
	"github.com/caddyserver/ingress/pkg/converter"
	"github.com/caddyserver/ingress/pkg/store"
	"sort"
	"strings"
)

type IngressSortPlugin struct{}

func (p IngressSortPlugin) IngressPlugin() converter.PluginInfo {
	return converter.PluginInfo{
		Name: "ingress_sort",
		// Must go after ingress are configured
		Priority: -2,
		New:      func() converter.Plugin { return new(IngressSortPlugin) },
	}
}

func init() {
	converter.RegisterPlugin(IngressSortPlugin{})
}

func getFirstItemFromJSON(data json.RawMessage) string {
	var arr []string
	err := json.Unmarshal(data, &arr)
	if err != nil {
		return ""
	}
	return arr[0]
}

// GlobalHandler in IngressSortPlugin tries to sort routes to have the less conflict.
//
// It only supports basic conflicts for now. It doesn't support multiple matchers in the same route
// nor multiple path/host in the matcher. It shouldn't be an issue with the ingress.matcher plugin.
// Sort will prioritize exact paths then prefix paths and finally empty paths.
// When 2 exacts paths or 2 prefixed paths are on the same host, we choose the longer first.
func (p IngressSortPlugin) GlobalHandler(config *converter.Config, store *store.Store) error {
	if !store.ConfigMap.ExperimentalSmartSort {
		return nil
	}

	routes := config.GetHTTPServer().Routes

	pathMap := make(map[int]string)
	for i, route := range routes {
		pathMap[i] = getFirstItemFromJSON(route.MatcherSetsRaw[0]["path"])
	}

	sort.SliceStable(routes, func(i, j int) bool {
		iPath := pathMap[i]
		jPath := pathMap[j]
		iPrefixed := strings.HasSuffix(iPath, "*")
		jPrefixed := strings.HasSuffix(jPath, "*")

		// If both same type check by length
		if iPrefixed == jPrefixed {
			return len(jPath) > len(iPath)
		}
		// Empty path will be moved last
		if jPath == "" || iPath == "" {
			return iPath == ""
		}
		// j path is exact so should go first
		return iPrefixed
	})

	return nil
}

// Interface guards
var (
	_ = converter.GlobalMiddleware(IngressSortPlugin{})
)
