package proxy

import (
	"net"

	"github.com/caddyserver/caddy/v2"
	"github.com/pires/go-proxyproto"
)

var (
	_ = caddy.Provisioner(&Wrapper{})
	_ = caddy.Module(&Wrapper{})
	_ = caddy.ListenerWrapper(&Wrapper{})
)

func init() {
	caddy.RegisterModule(Wrapper{})
}

// Wrapper provides PROXY protocol support to Caddy by implementing the caddy.ListenerWrapper interface.
// It must be loaded before the `tls` listener.
type Wrapper struct {
	policy proxyproto.PolicyFunc
}

func (Wrapper) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "caddy.listeners.proxy_protocol",
		New: func() caddy.Module { return new(Wrapper) },
	}
}

func (pp *Wrapper) Provision(ctx caddy.Context) error {
	pp.policy = func(upstream net.Addr) (proxyproto.Policy, error) {
		return proxyproto.REQUIRE, nil
	}
	return nil
}

func (pp *Wrapper) WrapListener(l net.Listener) net.Listener {
	pL := &proxyproto.Listener{Listener: l, Policy: pp.policy}

	return pL
}
