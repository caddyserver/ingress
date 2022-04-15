package coraza

import v1 "k8s.io/api/networking/v1"

const (
	annotationPrefix      = "caddy.ingress.kubernetes.io"
	enableWAF             = "enable-waf"
	modsecurityDirectives = "modsecurity-directives"
	enableOWASPCoreRules  = "enable-owasp-core-rules"
	modsecurityIncludes   = "modsecurity-includes"
)

func getAnnotation(ing *v1.Ingress, rule string) string {
	return ing.Annotations[annotationPrefix+"/"+rule]
}
