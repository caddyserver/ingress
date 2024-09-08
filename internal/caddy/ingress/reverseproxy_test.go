package ingress

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/ingress/pkg/converter"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTrustedProxesConvertToCaddyConfig(t *testing.T) {
	rpp := ReverseProxyPlugin{}

	tests := []struct {
		name               string
		annotations        map[string]string
		expectedConfigPath string
	}{
		{
			name: "ipv4 trusted proxies",
			annotations: map[string]string{
				"caddy.ingress.kubernetes.io/trusted-proxies": "192.168.1.0, 10.0.0.1",
			},
			expectedConfigPath: "test_data/reverseproxy_trusted_proxies_ipv4.json",
		},
		{
			name: "ipv6 trusted proxies",
			annotations: map[string]string{
				"caddy.ingress.kubernetes.io/trusted-proxies": "2001:db8::1, 2001:db8::5",
			},
			expectedConfigPath: "test_data/reverseproxy_trusted_proxies_ipv6.json",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := converter.IngressMiddlewareInput{
				Ingress: &networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: test.annotations,
						Namespace:   "namespace",
					},
				},
				Path: networkingv1.HTTPIngressPath{
					Backend: networkingv1.IngressBackend{
						Service: &networkingv1.IngressServiceBackend{
							Name: "svcName",
							Port: networkingv1.ServiceBackendPort{Number: 80},
						},
					},
				},
				Route: &caddyhttp.Route{},
			}

			route, err := rpp.IngressHandler(input)
			require.NoError(t, err)

			expectedCfg, err := os.ReadFile(test.expectedConfigPath)
			require.NoError(t, err)

			cfgJson, err := json.Marshal(&route)
			require.NoError(t, err)

			require.JSONEq(t, string(expectedCfg), string(cfgJson))
		})
	}
}

func TestMisconfiguredTrustedProxiesConvertToCaddyConfig(t *testing.T) {
	rpp := ReverseProxyPlugin{}

	tests := []struct {
		name          string
		annotations   map[string]string
		expectedError string
	}{
		{
			name: "invalid ipv4 trusted proxy",
			annotations: map[string]string{
				"caddy.ingress.kubernetes.io/trusted-proxies": "999.999.999.999",
			},
			expectedError: `failed to parse IP: "999.999.999.999"`,
		},
		{
			name: "invalid ipv6 trusted proxy",
			annotations: map[string]string{
				"caddy.ingress.kubernetes.io/trusted-proxies": "2001:db8::g",
			},
			expectedError: `failed to parse IP: "2001:db8::g"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := converter.IngressMiddlewareInput{
				Ingress: &networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: test.annotations,
						Namespace:   "namespace",
					},
				},
				Path: networkingv1.HTTPIngressPath{
					Backend: networkingv1.IngressBackend{
						Service: &networkingv1.IngressServiceBackend{
							Name: "svcName",
							Port: networkingv1.ServiceBackendPort{Number: 80},
						},
					},
				},
				Route: &caddyhttp.Route{},
			}

			route, err := rpp.IngressHandler(input)
			require.EqualError(t, err, test.expectedError)

			cfgJson, err := json.Marshal(&route)
			require.NoError(t, err)

			require.JSONEq(t, string(cfgJson), "null")
		})
	}
}
