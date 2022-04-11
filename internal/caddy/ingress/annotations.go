package ingress

import v1 "k8s.io/api/networking/v1"

const (
	annotationPrefix             = "caddy.ingress.kubernetes.io"
	rewriteToAnnotation          = "rewrite-to"
	rewriteStripPrefixAnnotation = "rewrite-strip-prefix"
	disableSSLRedirect           = "disable-ssl-redirect"
)

func getAnnotation(ing *v1.Ingress, rule string) string {
	return ing.Annotations[annotationPrefix+"/"+rule]
}
