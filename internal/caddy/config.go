package caddy

import (
	"encoding/json"

	"bitbucket.org/lightcodelabs/caddy2"
)

type serverRoute struct {
	Matchers  map[string]json.RawMessage `json:"match"`
	Apply     []map[string]string        `json:"apply"`
	Respond   proxyConfig                `json:"respond"`
	Exclusive bool                       `json:"exclusive"`
}

type routeList []serverRoute

type proxyConfig struct {
	Module          string `json:"_module"`
	LoadBalanceType string `json:"load_balance_type"`
	Upstreams       []upstreamConfig
}

type upstreamConfig struct {
	Host string `json:"host"`
}

type httpServerConfig struct {
	Listen            []string        `json:"listen"`
	ReadTimeout       caddy2.Duration `json:"read_timeout"`
	ReadHeaderTimeout caddy2.Duration `json:"read_header_timeout"`
	HiddenFiles       []string        `json:"hidden_files"` // TODO:... experimenting with shared/common state
	Routes            routeList       `json:"routes"`
	Errors            httpErrorConfig `json:"errors"`
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

type httpServer struct {
	HTTP servers `json:"http"`
}

// Config represents a caddy2 config file.
type Config struct {
	Modules httpServer `json:"modules"`
}

// NewConfig returns a plain slate caddy2 config file.
func NewConfig() *Config {
	return &Config{
		Modules: httpServer{
			HTTP: servers{
				Servers: serverConfig{
					Server: httpServerConfig{
						Listen: []string{":80", ":443"},
					},
				},
			},
		},
	}
}
