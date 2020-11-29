package caddy

import (
	"encoding/json"
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/ingress/internal/controller"
	"k8s.io/api/networking/v1beta1"
)

// TODO :- configure log middleware for all routes
func baseRoute(upstream string) caddyhttp.Route {
	return caddyhttp.Route{
		HandlersRaw: []json.RawMessage{
			json.RawMessage(`
			{
				"handler": "reverse_proxy",
				"upstreams": [
						{
								"dial": "` + fmt.Sprintf("%s", upstream) + `"
						}
				]
			}
		`),
		},
	}
}

// LoadIngressConfig creates a routelist based off of ingresses managed by this controller.
func LoadIngressConfig(config *Config, store *controller.Store) error {
	// TODO :-
	// when setting the upstream url we should should bypass kube-dns and get the ip address of
	// the pod for the deployment we are proxying to so that we can proxy to that ip address port.
	// this is good for session affinity and increases performance.

	// create a server route for each ingress route
	var routes caddyhttp.RouteList
	for _, ing := range store.Ingresses {
		for _, rule := range ing.Spec.Rules {
			for _, path := range rule.HTTP.Paths {
				clusterHostName := fmt.Sprintf("%v.%v.svc.cluster.local:%d", path.Backend.ServiceName, ing.Namespace, path.Backend.ServicePort.IntVal)
				r := baseRoute(clusterHostName)

				match := caddy.ModuleMap{}

				if rule.Host != "" {
					match["host"] = caddyconfig.JSON(caddyhttp.MatchHost{rule.Host}, nil)
				}

				if path.Path != "" {
					p := path.Path

					if *path.PathType == v1beta1.PathTypePrefix {
						p += "*"
					}
					match["path"] = caddyconfig.JSON(caddyhttp.MatchPath{p}, nil)
				}

				r.MatcherSetsRaw = []caddy.ModuleMap{match}

				routes = append(routes, r)
			}
		}
	}

	httpApp := config.Apps["http"].(*caddyhttp.App)
	httpApp.Servers[HttpServer].Routes = routes

	return nil
}
