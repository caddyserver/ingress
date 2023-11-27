package ingress

import v1 "k8s.io/api/networking/v1"

const (
	annotationPrefix             = "caddy.ingress.kubernetes.io"
	backendProtocol              = "backend-protocol"
	insecureSkipVerify           = "insecure-skip-verify"
	rewriteStripPrefixAnnotation = "rewrite-strip-prefix"
	rewriteToAnnotation          = "rewrite-to"
	sslRedirect                  = "ssl-redirect"

	//// Deprecated annotations

	// Use "ssl-redirect" instead, see https://github.com/caddyserver/ingress/issues/102
	disableSSLRedirect = "disable-ssl-redirect"
)

func getAnnotation(ing *v1.Ingress, rule string) string {
	return ing.Annotations[annotationPrefix+"/"+rule]
}

func getAnnotationBool(ing *v1.Ingress, rule string, def bool) bool {
	val := getAnnotation(ing, rule)
	if val == "" {
		return def
	}
	return val == "true"
}

func hasAnnotation(ing *v1.Ingress, rule string) bool {
	_, ok := ing.Annotations[annotationPrefix+"/"+rule]
	return ok
}
