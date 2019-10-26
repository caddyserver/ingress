package caddy

import (
	"encoding/json"
	"fmt"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"k8s.io/api/extensions/v1beta1"
)

// ConvertToCaddyConfig returns a new caddy routelist based off of ingresses managed by this controller.
// This is not used when this ingress controller is configured with a config map, so that we don't
// override user defined routes.
func ConvertToCaddyConfig(ings []*v1beta1.Ingress) (caddyhttp.RouteList, error) {
	// TODO :-
	// when setting the upstream url we should should bypass kube-dns and get the ip address of
	// the pod for the deployment we are proxying to so that we can proxy to that ip address port.
	// this is good for session affinity and increases performance.

	// create a server route for each ingress route
	var routes caddyhttp.RouteList
	for _, ing := range ings {
		for _, rule := range ing.Spec.Rules {
			for _, path := range rule.HTTP.Paths {
				clusterHostName := fmt.Sprintf("%v.%v.svc.cluster.local", path.Backend.ServiceName, ing.Namespace)
				r := baseRoute(clusterHostName)

				r.MatcherSets = caddyhttp.MatcherSets{
					{
						caddyhttp.MatchHost{rule.Host},
						caddyhttp.MatchPath{path.Path},
					},
				}

				routes = append(routes, r)
			}
		}
	}

	return routes, nil
}

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
