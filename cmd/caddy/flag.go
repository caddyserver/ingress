package main

import (
	"flag"

	"github.com/caddyserver/ingress/internal/caddy"
)

func parseFlags() caddy.ControllerConfig {
	var namespace string
	flag.StringVar(&namespace, "namespace", "", "the namespace that you would like to observe kubernetes ingress resources in.")

	var configMapName string
	flag.StringVar(&configMapName, "config-map", "", "defines the config map name from where to load global options")

	flag.Parse()

	return caddy.ControllerConfig{
		WatchNamespace: namespace,
		ConfigMapName:  configMapName,
	}
}
