package caddy

import (
	"encoding/json"

	"bitbucket.org/lightcodelabs/caddy2/modules/caddytls"
)

type serverRoute struct {
	Matchers map[string]json.RawMessage `json:"match"`
	Apply    []map[string]string        `json:"apply"`
	Respond  proxyConfig                `json:"respond"`
}

type routeList []serverRoute

type proxyConfig struct {
	Module          string           `json:"responder"`
	LoadBalanceType string           `json:"load_balance_type"`
	Upstreams       []upstreamConfig `json:"upstreams"`
}

type upstreamConfig struct {
	Host string `json:"host"`
}

type httpServerConfig struct {
	Listen           []string `json:"listen"`
	ReadTimeout      string   `json:"read_timeout"`
	DisableAutoHTTPS bool     `json:"disable_auto_https"`
	// ReadHeaderTimeout caddy2.Duration `json:"read_header_timeout"`
	// HiddenFiles []string  `json:"hidden_files"` // TODO:... experimenting with shared/common state
	TLSConnPolicies caddytls.ConnectionPolicies `json:"tls_connection_policies"`
	Routes          routeList                   `json:"routes"`
}

type httpErrorConfig struct {
	Routes routeList `json:"routes"`
}

type serverConfig struct {
	Server httpServerConfig `json:"ingress_server"`
}

type servers struct {
	Servers serverConfig `json:"servers"`
}

type TLSConfig struct {
	Module     string                    `json:"module"`
	Automation caddytls.AutomationConfig `json:"automation"`
}

type httpServer struct {
	TLS  TLSConfig `json:"tls"`
	HTTP servers   `json:"http"`
}

// StorageValues represents the config for certmagic storage providers.
type StorageValues struct {
	Namespace string `json:"namespace"`
}

// Storage represents the certmagic storage configuration.
type Storage struct {
	System string `json:"system"`
	StorageValues
}

// Config represents a caddy2 config file.
type Config struct {
	Storage Storage    `json:"storage"`
	Modules httpServer `json:"apps"`
}

// NewConfig returns a plain slate caddy2 config file.
func NewConfig(namespace string) *Config {
	// TODO :- get email from arguments to ingress controller
	autoPolicyBytes := json.RawMessage(`{"module": "acme", "email": "navdgo@gmail.com"}`)

	return &Config{
		Storage: Storage{
			System: "secret_store",
			StorageValues: StorageValues{
				Namespace: namespace,
			},
		},
		Modules: httpServer{
			TLS: TLSConfig{
				Module: "acme",
				Automation: caddytls.AutomationConfig{
					Policies: []caddytls.AutomationPolicy{
						caddytls.AutomationPolicy{
							Hosts:      nil,
							Management: autoPolicyBytes,
						},
					},
				},
			},
			HTTP: servers{
				Servers: serverConfig{
					Server: httpServerConfig{
						DisableAutoHTTPS: false, // TODO :- allow to be set from arguments to ingress controller
						ReadTimeout:      "30s",
						Listen:           []string{":80", ":443"},
						TLSConnPolicies: caddytls.ConnectionPolicies{
							&caddytls.ConnectionPolicy{},
						},
					},
				},
			},
		},
	}
}
