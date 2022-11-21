package ingress

import (
	v1 "k8s.io/api/networking/v1"
	"strconv"
)

const (
	annotationPrefix = "caddy.ingress.kubernetes.io"

	RewriteToAnnotation          = "rewrite-to"
	RewriteStripPrefixAnnotation = "rewrite-strip-prefix"
	DisableSSLRedirect           = "disable-ssl-redirect"
	BackendProtocol              = "backend-protocol"
	InsecureSkipVerify           = "insecure-skip-verify"
	OnDemandTLS                  = "on-demand-tls"
	OnDemandTLSAsk               = "on-demand-tls-ask"
	OnDemandTLSRateLimitInterval = "on-demand-tls-rate-limit-interval"
	OnDemandTLSRateLimitBurst    = "on-demand-tls-rate-limit-burst"
)

func GetAnnotation(ing *v1.Ingress, rule string) string {
	return ing.Annotations[annotationPrefix+"/"+rule]
}

func HasAnnotation(ing *v1.Ingress, rule string) bool {
	return ing.Annotations[annotationPrefix+"/"+rule] != ""
}

func GetAnnotationBool(ing *v1.Ingress, rule string, def bool) bool {
	val := GetAnnotation(ing, rule)
	if val == "" {
		return def
	}
	return val == "true"
}

func GetAnnotationInt(ing *v1.Ingress, rule string, def int) int {
	val := GetAnnotation(ing, rule)
	number, err := strconv.Atoi(val)
	if err != nil {
		return def
	}
	return number
}
