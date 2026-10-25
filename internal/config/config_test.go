package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	defaults := Config{
		HTTPAddr:       "127.0.0.1:8080",
		GRPCAddr:       "127.0.0.1:9090",
		BackendAddr:    "127.0.0.1:9092",
		BackendTimeout: 3 * time.Second,
	}
	tests := []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr string
	}{
		{
			name: "defaults",
			want: defaults,
		},
		{
			name: "overrides",
			env: map[string]string{
				"HTTP_ADDR":       "0.0.0.0:8081",
				"GRPC_ADDR":       "[::1]:9091",
				"BACKEND_ADDR":    "catalog:9092",
				"BACKEND_TIMEOUT": "250ms",
			},
			want: Config{
				HTTPAddr:       "0.0.0.0:8081",
				GRPCAddr:       "[::1]:9091",
				BackendAddr:    "catalog:9092",
				BackendTimeout: 250 * time.Millisecond,
			},
		},
		{
			name:    "HTTP address without port",
			env:     map[string]string{"HTTP_ADDR": "localhost"},
			wantErr: "HTTP_ADDR",
		},
		{
			name:    "invalid timeout",
			env:     map[string]string{"BACKEND_TIMEOUT": "soon"},
			wantErr: "BACKEND_TIMEOUT",
		},
		{
			name:    "zero timeout",
			env:     map[string]string{"BACKEND_TIMEOUT": "0s"},
			wantErr: "BACKEND_TIMEOUT",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Isolate defaults from the developer's environment; Setenv restores it at cleanup.
			for _, key := range []string{"HTTP_ADDR", "GRPC_ADDR", "BACKEND_ADDR", "BACKEND_TIMEOUT"} {
				t.Setenv(key, "")
				if !assert.NoError(t, os.Unsetenv(key)) {
					return
				}
			}
			for key, value := range tt.env {
				t.Setenv(key, value)
			}
			got, err := Load()
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				assert.Empty(t, got)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
