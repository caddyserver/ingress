package ingress

import v1 "k8s.io/api/networking/v1"

const (
	annotationPrefix                = "caddy.ingress.kubernetes.io"
	rewriteToAnnotation             = "rewrite-to"
	rewriteStripPrefixAnnotation    = "rewrite-strip-prefix"
	disableSSLRedirect              = "disable-ssl-redirect"
	backendProtocol                 = "backend-protocol"
	insecureSkipVerify              = "insecure-skip-verify"
	permanentRedirectAnnotation     = "permanent-redirect"
	permanentRedirectCodeAnnotation = "permanent-redirect-code"
	temporaryRedirectAnnotation     = "temporal-redirect"
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
