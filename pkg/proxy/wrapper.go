package proxy

import (
	"net"

	"github.com/caddyserver/caddy/v2"
	"github.com/pires/go-proxyproto"
)

// Wrapper provides PROXY protocol support to Caddy by implementing the caddy.ListenerWrapper interface. It must be loaded before the `tls` listener.
type Wrapper struct {
	// Allow is an optional list of CIDR ranges to allow/require PROXY headers from.
	Allow []string `json:"allow,omitempty"`

	policy proxyproto.PolicyFunc
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
