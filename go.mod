module github.com/caddyserver/ingress

go 1.16

require (
	github.com/caddyserver/caddy/v2 v2.4.6
	github.com/caddyserver/certmagic v0.15.2
	github.com/corazawaf/coraza-caddy v1.2.0
	github.com/google/uuid v1.3.0
	github.com/mitchellh/mapstructure v1.4.3
	github.com/pires/go-proxyproto v0.3.1
	github.com/pkg/errors v0.9.1
	go.uber.org/zap v1.21.0
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/go-playground/pool.v3 v3.1.1
	k8s.io/api v0.19.4
	k8s.io/apimachinery v0.19.4
	k8s.io/client-go v0.19.4
)

replace (
	k8s.io/api => k8s.io/api v0.19.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.4
	k8s.io/client-go => k8s.io/client-go v0.19.4
)
