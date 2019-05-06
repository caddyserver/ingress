package caddy

import (
	"encoding/json"
	"fmt"

	"k8s.io/api/extensions/v1beta1"
)

// ConvertToCaddyConfig returns a new caddy routelist based off of ingresses managed by this controller.
func ConvertToCaddyConfig(ings []*v1beta1.Ingress) ([]serverRoute, []string, error) {
	// ~~~~
	// TODO :-
	// when setting the upstream url we should should bypass kube-proxy and get the ip address of
	// the pod for the deployment we are proxying to so that we can proxy to that ip address port.
	// this is good for session affinity and increases performance (since we don't have to hit dns).
	// ~~~~

	// record hosts for tls policies
	var hosts []string

	// create a server route for each ingress route
	var routes routeList
	for _, ing := range ings {
		for _, rule := range ing.Spec.Rules {
			hosts = append(hosts, rule.Host)

			for _, path := range rule.HTTP.Paths {
				clusterHostName := fmt.Sprintf("%v.%v.svc.cluster.local", path.Backend.ServiceName, ing.Namespace)
				r := baseRoute(clusterHostName)

				// create matchers for ingress host and path
				h := json.RawMessage(fmt.Sprintf(`["%v"]`, rule.Host))
				p := json.RawMessage(fmt.Sprintf(`["%v"]`, path.Path))

				r.Matchers = map[string]json.RawMessage{
					"host": h,
					"path": p,
				}

				// add logging middleware to all routes
				r.Apply = []map[string]string{
					map[string]string{
						"file":       "access.log",
						"middleware": "log",
					},
				}

				routes = append(routes, r)
			}
		}
	}

	return routes, hosts, nil
}

func baseRoute(upstream string) serverRoute {
	return serverRoute{
		Apply: []map[string]string{
			map[string]string{
				"middleware": "log",
				"file":       "access.log",
			},
		},
		Respond: proxyConfig{
			Module:          "reverse_proxy",
			LoadBalanceType: "random",
			Upstreams: []upstreamConfig{
				upstreamConfig{
					Host: fmt.Sprintf("http://%v", upstream),
				},
			},
		},
	}
}
