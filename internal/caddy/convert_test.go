package caddy

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/caddyserver/ingress/pkg/store"
	"github.com/stretchr/testify/require"
)

func TestConvertToCaddyConfig(t *testing.T) {
	tests := []struct {
		name               string
		expectedConfigPath string
	}{
		{
			name:               "default",
			expectedConfigPath: "./test_data/default.json",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store, err := store.NewStore(nil, nil, store.Options{}, "", &store.PodInfo{}, nil, nil, nil, nil)
			require.NoError(t, err)

			cfg, err := Converter{}.ConvertToCaddyConfig(store)
			require.NoError(t, err)

			cfgJSON, err := json.Marshal(cfg)
			require.NoError(t, err)

			expectedCfg, err := os.ReadFile(test.expectedConfigPath)
			require.NoError(t, err)

			require.JSONEq(t, string(expectedCfg), string(cfgJSON))
		})
	}
}
