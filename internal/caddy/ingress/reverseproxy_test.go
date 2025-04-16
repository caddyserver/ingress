package ingress

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/ingress/pkg/converter"
	"github.com/caddyserver/ingress/pkg/store"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/cache"
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
			name: "ipv4 trusted proxies wit subnet",
			annotations: map[string]string{
				"caddy.ingress.kubernetes.io/trusted-proxies": "192.168.1.0/16,10.0.0.1/8",
			},
			expectedConfigPath: "test_data/reverseproxy_trusted_proxies_ipv4_subnet.json",
		},
		{
			name: "ipv6 trusted proxies",
			annotations: map[string]string{
				"caddy.ingress.kubernetes.io/trusted-proxies": "2001:db8::1, 2001:db8::5",
			},
			expectedConfigPath: "test_data/reverseproxy_trusted_proxies_ipv6.json",
		},
		{
			name: "ipv6 trusted proxies",
			annotations: map[string]string{
				"caddy.ingress.kubernetes.io/trusted-proxies": "2001:db8::1/36,2001:db8::5/60",
			},
			expectedConfigPath: "test_data/reverseproxy_trusted_proxies_ipv6_subnet.json",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := prepareReverseProxyTestInput()
			input.Ingress.ObjectMeta.Annotations = test.annotations

			route, err := rpp.IngressHandler(input)
			require.NoError(t, err)

			expectedCfg, err := os.ReadFile(test.expectedConfigPath)
			require.NoError(t, err)

			cfgJSON, err := json.Marshal(&route)
			require.NoError(t, err)

			require.JSONEq(t, string(expectedCfg), string(cfgJSON))
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
			name: "invalid ipv4 with subnet trusted proxy",
			annotations: map[string]string{
				"caddy.ingress.kubernetes.io/trusted-proxies": "999.999.999.999/32",
			},
			expectedError: `failed to parse IP: "999.999.999.999/32"`,
		},
		{
			name: "invalid subnet for ipv4 trusted proxy",
			annotations: map[string]string{
				"caddy.ingress.kubernetes.io/trusted-proxies": "10.0.0.0/100",
			},
			expectedError: `failed to parse IP: "10.0.0.0/100"`,
		},
		{
			name: "invalid ipv6 trusted proxy",
			annotations: map[string]string{
				"caddy.ingress.kubernetes.io/trusted-proxies": "2001:db8::g",
			},
			expectedError: `failed to parse IP: "2001:db8::g"`,
		},
		{
			name: "invalid ipv6 with subnet trusted proxy",
			annotations: map[string]string{
				"caddy.ingress.kubernetes.io/trusted-proxies": "2001:db8::g/128",
			},
			expectedError: `failed to parse IP: "2001:db8::g/128"`,
		},
		{
			name: "invalid subnet for ipv6 trusted proxy",
			annotations: map[string]string{
				"caddy.ingress.kubernetes.io/trusted-proxies": "2001:db8::/200",
			},
			expectedError: `failed to parse IP: "2001:db8::/200"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := prepareReverseProxyTestInput()
			input.Ingress.ObjectMeta.Annotations = test.annotations

			route, err := rpp.IngressHandler(input)
			require.EqualError(t, err, test.expectedError)

			cfgJSON, err := json.Marshal(&route)
			require.NoError(t, err)

			require.JSONEq(t, string(cfgJSON), "null")
		})
	}
}

func prepareReverseProxyTestInput() converter.IngressMiddlewareInput {
	serviceCache := cache.NewIndexer(cache.MetaNamespaceKeyFunc, make(cache.Indexers))
	endpointSliceCache := cache.NewIndexer(cache.MetaNamespaceKeyFunc, make(cache.Indexers))
	store, _ := store.NewStore(nil, nil, store.Options{}, "", nil, nil, serviceCache, endpointSliceCache, nil)

	serviceCache.Add(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
			Name:      "svcName",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{Port: 80, TargetPort: intstr.FromInt(8080)},
			},
		},
	})
	endpointSliceCache.Add(&discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
			Name:      "svcName-1234",
			Labels: map[string]string{
				discoveryv1.LabelServiceName: "svcName",
			},
		},
		Endpoints: []discoveryv1.Endpoint{
			{Addresses: []string{"10.20.30.40"}},
		},
	})

	return converter.IngressMiddlewareInput{
		Store: store,
		Ingress: &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "namespace",
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
}
