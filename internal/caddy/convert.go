package caddy

import (
	"encoding/json"
	"fmt"

	"github.com/caddyserver/caddy2/modules/caddyhttp"
	"k8s.io/api/extensions/v1beta1"
)

// ConvertToCaddyConfig returns a new caddy routelist based off of ingresses managed by this controller.
// This is not used when this ingress controller is configured with a config map, so that we don't
// override user defined routes.
func ConvertToCaddyConfig(ings []*v1beta1.Ingress) (caddyhttp.RouteList, error) {
	// ~~~~
	// TODO :-
	// when setting the upstream url we should should bypass kube-dns and get the ip address of
	// the pod for the deployment we are proxying to so that we can proxy to that ip address port.
	// this is good for session affinity and increases performance.
	// ~~~~

	// create a server route for each ingress route
	var routes caddyhttp.RouteList
	for _, ing := range ings {
		for _, rule := range ing.Spec.Rules {
			for _, path := range rule.HTTP.Paths {
				clusterHostName := fmt.Sprintf("%v.%v.svc.cluster.local", path.Backend.ServiceName, ing.Namespace)
				r := baseRoute(clusterHostName)

				// create matchers for ingress host and path
				h := json.RawMessage(fmt.Sprintf(`["%v"]`, rule.Host))
				p := json.RawMessage(fmt.Sprintf(`["%v"]`, path.Path))

				r.MatcherSets = []map[string]json.RawMessage{
					{
						"host": h,
						"path": p,
					},
				}

				routes = append(routes, r)
			}
		}
	}

	return routes, nil
}

func baseRoute(upstream string) caddyhttp.ServerRoute {
	return caddyhttp.ServerRoute{
		Apply: []json.RawMessage{
			json.RawMessage(`
				{
					"middleware": "log",
					"filename":   "/etc/caddy/access.log"
				}
			`),
		},
		Respond: json.RawMessage(`
			{
				"responder": "reverse_proxy",
				"load_balance_type": "random",
				"upstreams": [
						{
								"host": "` + fmt.Sprintf("http://%v", upstream) + `"
						}
				]
			}
		`),
	}
}
