package main

import (
	"flag"

	"github.com/caddyserver/ingress/internal/caddy"
	"github.com/sirupsen/logrus"
)

func parseFlags() caddy.ControllerConfig {
	var email string
	flag.StringVar(&email, "email", "", "the email address to use for requesting tls certificates if automatic https is enabled.")

	var namespace string
	flag.StringVar(&namespace, "namespace", "", "the namespace that you would like to observe kubernetes ingress resources in.")

	var enableAutomaticTLS bool
	flag.BoolVar(&enableAutomaticTLS, "tls", false, "defines if automatic tls should be enabled for hostnames defined in ingress resources.")

	var tlsUseStaging bool
	flag.BoolVar(&tlsUseStaging, "tls-use-staging", false, "defines if the lets-encrypt staging server should be used for testing the provisioning of tls certificates.")

	var automaticRedirects bool
	flag.BoolVar(&automaticRedirects, "automatic-redirects", true, "defines if automatic HTTP -> HTTPS redirects should be enabled.")

	flag.Parse()

	if email == "" && enableAutomaticTLS {
		logrus.Info("An email must be defined for automatic tls features, set flag `email` with the email address you would like to use for certificate registration.")
		enableAutomaticTLS = false
	}

	return caddy.ControllerConfig{
		Email:              email,
		AutomaticTLS:       enableAutomaticTLS,
		AutomaticRedirects: automaticRedirects,
		TLSUseStaging:      tlsUseStaging,
		WatchNamespace:     namespace,
	}
}
