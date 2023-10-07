package global

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func TestIngressSort(t *testing.T) {
	tests := []struct {
		name   string
		routes []struct {
			id   int
			path string
		}
		expect []int
	}{

		{
			name: "multiple exact paths",
			routes: []struct {
				id   int
				path string
			}{
				{id: 0, path: "/path/a"},
				{id: 1, path: "/path/"},
				{id: 2, path: "/other"},
			},
			expect: []int{0, 1, 2},
		},
		{
			name: "multiple prefix paths",
			routes: []struct {
				id   int
				path string
			}{
				{id: 0, path: "/path/*"},
				{id: 1, path: "/path/auth/*"},
				{id: 2, path: "/other/*"},
				{id: 3, path: "/login/*"},
			},
			expect: []int{1, 2, 3, 0},
		},
		{
			name: "mixed exact and prefixed",
			routes: []struct {
				id   int
				path string
			}{
				{id: 0, path: "/path/*"},
				{id: 1, path: "/path/auth/"},
				{id: 2, path: "/path/v2/*"},
				{id: 3, path: "/path/new"},
			},
			expect: []int{1, 3, 2, 0},
		},
		{
			name: "mixed exact, prefix and empty",
			routes: []struct {
				id   int
				path string
			}{
				{id: 0, path: "/path/*"},
				{id: 1, path: ""},
				{id: 2, path: "/path/v2/*"},
				{id: 3, path: "/path/new"},
				{id: 4, path: ""},
			},
			expect: []int{3, 2, 0, 1, 4},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			routes := []caddyhttp.Route{}

			for _, route := range test.routes {
				match := caddy.ModuleMap{}
				match["id"] = caddyconfig.JSON(route.id, nil)

				if route.path != "" {
					match["path"] = caddyconfig.JSON(caddyhttp.MatchPath{route.path}, nil)
				}

				r := caddyhttp.Route{MatcherSetsRaw: []caddy.ModuleMap{match}}
				routes = append(routes, r)
			}

			sortRoutes(routes)

			var got []int
			for i := range test.expect {
				var currentId int
				err := json.Unmarshal(routes[i].MatcherSetsRaw[0]["id"], &currentId)
				if err != nil {
					t.Fatalf("error unmarshaling id for i %v, %v", i, err)
				}
				got = append(got, currentId)
			}

			if !reflect.DeepEqual(test.expect, got) {
				t.Errorf("expected order to match: got %v, expected %v, %s", got, test.expect, routes[1].MatcherSetsRaw)
			}
		})
	}
}
