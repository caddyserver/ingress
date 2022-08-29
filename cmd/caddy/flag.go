package main

import (
	"flag"
	"strings"

	"github.com/caddyserver/ingress/pkg/store"
)

func parseFlags() store.Options {
	var namespace string
	flag.StringVar(&namespace, "namespace", "", "the namespace that you would like to observe kubernetes ingress resources in.")

	var className string
	flag.StringVar(&className, "class-name", "caddy", "class name of the ingress controller")

	var classNameRequired bool
	flag.BoolVar(&classNameRequired, "class-name-required", false, "only allow ingress resources with a matching ingress class name")

	var configMapName string
	flag.StringVar(&configMapName, "config-map", "", "defines the config map name from where to load global options")

	var leaseId string
	flag.StringVar(&leaseId, "lease-id", "", "defines the id of this instance for certmagic lock")

	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "set the log level to debug")

	var pluginsOrder string
	flag.StringVar(&pluginsOrder, "plugins-order", "", "defines the order plugins should be used")

	flag.Parse()

	return store.Options{
		WatchNamespace:    namespace,
		ClassName:         className,
		ClassNameRequired: classNameRequired,
		ConfigMapName:     configMapName,
		Verbose:           verbose,
		LeaseId:           leaseId,
		PluginsOrder:      strings.Split(pluginsOrder, ","),
	}
}
