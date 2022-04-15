package converter

import "testing"

func TestSortPlugins(t *testing.T) {
	tests := []struct {
		name    string
		order   []string
		plugins []PluginInfo
		expect  []string
	}{
		{
			name:    "default to alpha sort",
			order:   nil,
			plugins: []PluginInfo{{Name: "b"}, {Name: "c"}, {Name: "a"}},
			expect:  []string{"a", "b", "c"},
		},
		{
			name:    "use priority when specified",
			order:   nil,
			plugins: []PluginInfo{{Name: "b"}, {Name: "a", Priority: 20}, {Name: "c", Priority: 10}},
			expect:  []string{"a", "c", "b"},
		},
		{
			name:    "fallback to alpha when no priority",
			order:   nil,
			plugins: []PluginInfo{{Name: "b"}, {Name: "a"}, {Name: "c", Priority: 20}},
			expect:  []string{"c", "a", "b"},
		},
		{
			name:    "specify order",
			order:   []string{"c"},
			plugins: []PluginInfo{{Name: "b"}, {Name: "a"}, {Name: "c"}},
			expect:  []string{"c", "a", "b"},
		},
		{
			name:    "order overrides other settings",
			order:   []string{"c"},
			plugins: []PluginInfo{{Name: "b", Priority: 10}, {Name: "a"}, {Name: "c"}},
			expect:  []string{"c", "b", "a"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sortPlugins(test.plugins, test.order)
			for i, plugin := range test.plugins {
				if test.expect[i] != plugin.Name {
					t.Errorf("expected order to match %v: got %v, expected %v", test.expect, plugin.Name, test.expect[i])
				}
			}
		})
	}
}
