package proxy

import "github.com/caddyserver/caddy/v2"

var (
	_ = caddy.Provisioner(&Wrapper{})
	_ = caddy.Module(&Wrapper{})
	_ = caddy.ListenerWrapper(&Wrapper{})
)

func init() {
	caddy.RegisterModule(Wrapper{})
}

func (Wrapper) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "caddy.listeners.proxy_protocol",
		New: func() caddy.Module { return new(Wrapper) },
	}
}
