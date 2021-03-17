package caddy

import (
	"encoding/json"
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/rewrite"
	"github.com/caddyserver/ingress/internal/controller"
	"k8s.io/api/networking/v1beta1"
)

const (
	annotationPrefix             = "caddy.ingress.kubernetes.io"
	rewriteToAnnotation          = "rewrite-to"
	rewriteStripPrefixAnnotation = "rewrite-strip-prefix"
	disableSSLRedirect           = "disable-ssl-redirect"
)

func getAnnotation(ing *v1beta1.Ingress, rule string) string {
	return ing.Annotations[annotationPrefix+"/"+rule]
}

// TODO :- configure log middleware for all routes
func generateRoute(ing *v1beta1.Ingress, rule v1beta1.IngressRule, path v1beta1.HTTPIngressPath) caddyhttp.Route {
	var handlers []json.RawMessage

	// Generate handlers
	rewriteTo := getAnnotation(ing, rewriteToAnnotation)
	if rewriteTo != "" {
		handlers = append(handlers, caddyconfig.JSONModuleObject(
			rewrite.Rewrite{URI: rewriteTo},
			"handler", "rewrite", nil,
		))
	}

	rewriteStripPrefix := getAnnotation(ing, rewriteStripPrefixAnnotation)
	if rewriteStripPrefix != "" {
		handlers = append(handlers, caddyconfig.JSONModuleObject(
			rewrite.Rewrite{StripPathPrefix: rewriteStripPrefix},
			"handler", "rewrite", nil,
		))
	}

	clusterHostName := fmt.Sprintf("%v.%v.svc.cluster.local:%d", path.Backend.ServiceName, ing.Namespace, path.Backend.ServicePort.IntVal)
	handlers = append(handlers, caddyconfig.JSONModuleObject(
		reverseproxy.Handler{
			Upstreams: reverseproxy.UpstreamPool{
				{Dial: clusterHostName},
			},
		},
		"handler", "reverse_proxy", nil,
	))

	// Generate matchers
	match := caddy.ModuleMap{}

	if getAnnotation(ing, disableSSLRedirect) != "true" {
		match["protocol"] = caddyconfig.JSON(caddyhttp.MatchProtocol("https"), nil)
	}

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

	return caddyhttp.Route{
		HandlersRaw:    handlers,
		MatcherSetsRaw: []caddy.ModuleMap{match},
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
				r := generateRoute(ing, rule, path)

				routes = append(routes, r)
			}
		}
	}

	httpApp := config.Apps["http"].(*caddyhttp.App)
	httpApp.Servers[HttpServer].Routes = routes

	return nil
}
