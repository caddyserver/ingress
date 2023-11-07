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
		expectedError      string
		annotations        map[string]string
	}{
		{
			name:               "Check permanent redirect without any specific redirect code",
			expectedConfigPath: "test_data/redirect_default.json",
			annotations: map[string]string{
				"caddy.ingress.kubernetes.io/permanent-redirect": "http://example.com",
			},
		},
		{
			name:               "Check permanent redirect with 'permanent' redirect code",
			expectedConfigPath: "test_data/redirect_permanent.json",
			annotations: map[string]string{
				"caddy.ingress.kubernetes.io/permanent-redirect":      "http://example.com",
				"caddy.ingress.kubernetes.io/permanent-redirect-code": "permanent",
			},
		},
		{
			name:               "Check permanent redirect with 'temporary' redirect code",
			expectedConfigPath: "test_data/redirect_temporary.json",
			annotations: map[string]string{
				"caddy.ingress.kubernetes.io/permanent-redirect":      "http://example.com",
				"caddy.ingress.kubernetes.io/permanent-redirect-code": "temporary",
			},
		},
		{
			name:               "Check permanent redirect with custom redirect code",
			expectedConfigPath: "test_data/redirect_custom_code.json",
			annotations: map[string]string{
				"caddy.ingress.kubernetes.io/permanent-redirect":      "http://example.com",
				"caddy.ingress.kubernetes.io/permanent-redirect-code": "308",
			},
		},
		{
			name:               "Check permanent redirect with invalid custom redirect code",
			expectedConfigPath: "",
			annotations: map[string]string{
				"caddy.ingress.kubernetes.io/permanent-redirect":      "http://example.com",
				"caddy.ingress.kubernetes.io/permanent-redirect-code": "502",
			},
			expectedError: "redir code not in the 3xx range or 401: '502'",
		},
		{
			name:               "Check permanent redirect with invalid custom redirect code string",
			expectedConfigPath: "",
			annotations: map[string]string{
				"caddy.ingress.kubernetes.io/permanent-redirect":      "http://example.com",
				"caddy.ingress.kubernetes.io/permanent-redirect-code": "randomstring",
			},
			expectedError: "not a supported redir code type or not valid integer: 'randomstring'",
		},
		{
			name:               "Check permanent redirect with 401 as redirect code",
			expectedConfigPath: "test_data/redirect_401.json",
			annotations: map[string]string{
				"caddy.ingress.kubernetes.io/permanent-redirect":      "http://example.com",
				"caddy.ingress.kubernetes.io/permanent-redirect-code": "401",
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

			route, err := rp.IngressHandler(input)
			if test.expectedError != "" {
				if assert.Error(t, err) {
					assert.EqualError(t, err, test.expectedError)
				}
			} else {
				assert.NoError(t, err, "unable to generate the route.")
			}

			if test.expectedError == "" {
				expectedCfg, err := os.ReadFile(test.expectedConfigPath)
				assert.NoError(t, err, "unable to find the file for comparison.")

				cfgJson, err := json.Marshal(&route)
				assert.NoError(t, err, "unable to marshal the route to JSON.")

				assert.JSONEq(t, string(cfgJson), string(expectedCfg))
			}
		})
	}
}
