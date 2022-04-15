package main

import (
	"flag"
	"github.com/caddyserver/ingress/pkg/store"
	"strings"
)

func parseFlags() store.Options {
	var namespace string
	flag.StringVar(&namespace, "namespace", "", "the namespace that you would like to observe kubernetes ingress resources in.")

	var configMapName string
	flag.StringVar(&configMapName, "config-map", "", "defines the config map name from where to load global options")

	var leaseId string
	flag.StringVar(&leaseId, "lease-id", "", "defines the id of this instance for certmagic lock")

	var verbose bool
	flag.BoolVar(&verbose, "v", false, "set the log level to debug")

	var pluginsOrder string
	flag.StringVar(&pluginsOrder, "plugins-order", "", "defines the order plugins should be used")

	flag.Parse()

	return store.Options{
		WatchNamespace: namespace,
		ConfigMapName:  configMapName,
		Verbose:        verbose,
		LeaseId:        leaseId,
		PluginsOrder:   strings.Split(pluginsOrder, ","),
	}
}
