package ingress

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/ingress/pkg/converter"
	"github.com/stretchr/testify/assert"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConvertToCaddyConfig(t *testing.T) {
	rp := RedirectPlugin{}

	tests := []struct {
		name               string
		expectedConfigPath string
		annotations        map[string]string
	}{
		{
			name:               "Cherk permanent redirect without any specific redirect code",
			expectedConfigPath: "test_data/redirect_default.json",
			annotations: map[string]string{
				"caddy.ingress.kubernetes.io/permanent-redirect": "http://example.com",
			},
		},
		{
			name:               "Cherk permanent redirect with 'permanent' redirect code",
			expectedConfigPath: "test_data/redirect_permanent.json",
			annotations: map[string]string{
				"caddy.ingress.kubernetes.io/permanent-redirect":      "http://example.com",
				"caddy.ingress.kubernetes.io/permanent-redirect-code": "permanent",
			},
		},
		{
			name:               "Cherk permanent redirect with 'temporary' redirect code",
			expectedConfigPath: "test_data/redirect_temporary.json",
			annotations: map[string]string{
				"caddy.ingress.kubernetes.io/permanent-redirect":      "http://example.com",
				"caddy.ingress.kubernetes.io/permanent-redirect-code": "temporary",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := converter.IngressMiddlewareInput{
				Ingress: &networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: test.annotations,
					},
				},
				Route: &caddyhttp.Route{},
			}

			expectedCfg, err := os.ReadFile(test.expectedConfigPath)
			assert.NoError(t, err, "unable to find the file for comparison.")

			route, err := rp.IngressHandler(input)
			assert.NoError(t, err, "unable to generaete the route.")

			cfgJson, err := json.Marshal(&route)
			assert.NoError(t, err, "unable to marshal the route to JSON.")

			assert.JSONEq(t, string(cfgJson), string(expectedCfg))
		})
	}
}
