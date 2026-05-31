package ingress

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/ingress/pkg/converter"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func pathType(t networkingv1.PathType) *networkingv1.PathType {
	return &t
}

func TestPathMatcherConvertToCaddyConfig(t *testing.T) {
	mp := MatcherPlugin{}

	tests := []struct {
		name         string
		path         string
		pathType     *networkingv1.PathType
		expectedPath string
	}{
		{
			name:         "prefix path matches the segment and its descendants only",
			path:         "/foo",
			pathType:     pathType(networkingv1.PathTypePrefix),
			expectedPath: `["/foo","/foo/*"]`,
		},
		{
			name:         "prefix path with a trailing slash is normalized",
			path:         "/foo/",
			pathType:     pathType(networkingv1.PathTypePrefix),
			expectedPath: `["/foo","/foo/*"]`,
		},
		{
			name:         "root prefix path emits Caddy match-all pattern",
			path:         "/",
			pathType:     pathType(networkingv1.PathTypePrefix),
			expectedPath: `["/*"]`,
		},
		{
			name:         "exact path matches verbatim",
			path:         "/foo",
			pathType:     pathType(networkingv1.PathTypeExact),
			expectedPath: `["/foo"]`,
		},
		{
			name:         "implementation specific path matches verbatim",
			path:         "/foo",
			pathType:     pathType(networkingv1.PathTypeImplementationSpecific),
			expectedPath: `["/foo"]`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := converter.IngressMiddlewareInput{
				Ingress: &networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"caddy.ingress.kubernetes.io/disable-ssl-redirect": "true",
						},
					},
				},
				Path: networkingv1.HTTPIngressPath{
					Path:     test.path,
					PathType: test.pathType,
				},
				Route: &caddyhttp.Route{},
			}

			route, err := mp.IngressHandler(input)
			require.NoError(t, err)
			require.Len(t, route.MatcherSetsRaw, 1)

			pathRaw, ok := route.MatcherSetsRaw[0]["path"]
			require.True(t, ok, "expected a path matcher")

			require.JSONEq(t, test.expectedPath, string(pathRaw))
		})
	}
}

func TestPrefixPathDoesNotMatchSiblingPrefix(t *testing.T) {
	mp := MatcherPlugin{}

	input := converter.IngressMiddlewareInput{
		Ingress: &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"caddy.ingress.kubernetes.io/disable-ssl-redirect": "true",
				},
			},
		},
		Path: networkingv1.HTTPIngressPath{
			Path:     "/foo",
			PathType: pathType(networkingv1.PathTypePrefix),
		},
		Route: &caddyhttp.Route{},
	}

	route, err := mp.IngressHandler(input)
	require.NoError(t, err)

	var matchPath caddyhttp.MatchPath
	require.NoError(t, json.Unmarshal(route.MatcherSetsRaw[0]["path"], &matchPath))

	cases := []struct {
		path string
		want bool
	}{
		{"/foo", true},
		{"/foo/", true},
		{"/foo/bar", true},
		{"/foobar", false},
		{"/foo-bar", false},
	}

	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, c.path, nil)
		repl := caddy.NewReplacer()
		req = req.WithContext(context.WithValue(req.Context(), caddy.ReplacerCtxKey, repl))

		got, err := matchPath.MatchWithError(req)
		require.NoError(t, err)
		require.Equalf(t, c.want, got, "path %q", c.path)
	}
}
