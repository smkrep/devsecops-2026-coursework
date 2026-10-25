package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	defaults := Config{
		HTTPAddr: "127.0.0.1:8080",
		GRPCAddr: "127.0.0.1:9090",
		CatalogBackendAddrs: []string{
			"127.0.0.1:9092",
		},
		OrdersBackendAddrs: []string{
			"127.0.0.1:9093",
		},
		BackendTimeout: 3 * time.Second,
		AuthBackendAddrs: []string{
			"127.0.0.1:9096",
		},
		AuthTimeout: time.Second,
	}
	tests := []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr error
	}{
		{
			name: "defaults",
			want: defaults,
		},
		{
			name: "overrides",
			env: map[string]string{
				"HTTP_ADDR":             "0.0.0.0:8081",
				"GRPC_ADDR":             "[::1]:9091",
				"CATALOG_BACKEND_ADDRS": " catalog:9092, [::1]:9094, catalog:9092 ",
				"ORDERS_BACKEND_ADDRS":  "orders:9093,orders:9095",
				"BACKEND_TIMEOUT":       "250ms",
				"AUTH_BACKEND_ADDRS":    "auth:9096,auth:9097",
				"AUTH_TIMEOUT":          "500ms",
			},
			want: Config{
				HTTPAddr: "0.0.0.0:8081",
				GRPCAddr: "[::1]:9091",
				CatalogBackendAddrs: []string{
					"catalog:9092",
					"[::1]:9094",
				},
				OrdersBackendAddrs: []string{
					"orders:9093",
					"orders:9095",
				},
				BackendTimeout: 250 * time.Millisecond,
				AuthBackendAddrs: []string{
					"auth:9096",
					"auth:9097",
				},
				AuthTimeout: 500 * time.Millisecond,
			},
		},
		{
			name: "HTTP address without port",
			env: map[string]string{
				"HTTP_ADDR": "localhost",
			},
			wantErr: ErrInvalidAddress,
		},
		{
			name: "orders address without port",
			env: map[string]string{
				"ORDERS_BACKEND_ADDRS": "orders",
			},
			wantErr: ErrInvalidAddress,
		},
		{
			name: "empty backend list",
			env: map[string]string{
				"CATALOG_BACKEND_ADDRS": "",
			},
			wantErr: ErrInvalidAddress,
		},
		{
			name: "empty address in list",
			env: map[string]string{
				"CATALOG_BACKEND_ADDRS": "catalog:9092,",
			},
			wantErr: ErrInvalidAddress,
		},
		{
			name: "invalid backend port",
			env: map[string]string{
				"ORDERS_BACKEND_ADDRS": "orders:70000",
			},
			wantErr: ErrInvalidAddress,
		},
		{
			name: "backend URI instead of address",
			env: map[string]string{
				"CATALOG_BACKEND_ADDRS": "http://catalog:9092",
			},
			wantErr: ErrInvalidAddress,
		},
		{
			name: "invalid auth address",
			env: map[string]string{
				"AUTH_BACKEND_ADDRS": "auth",
			},
			wantErr: ErrInvalidAddress,
		},
		{
			name: "invalid auth timeout",
			env: map[string]string{
				"AUTH_TIMEOUT": "soon",
			},
			wantErr: ErrInvalidTimeout,
		},
		{
			name: "zero auth timeout",
			env: map[string]string{
				"AUTH_TIMEOUT": "0s",
			},
			wantErr: ErrInvalidTimeout,
		},
		{
			name: "invalid timeout",
			env: map[string]string{
				"BACKEND_TIMEOUT": "soon",
			},
			wantErr: ErrInvalidTimeout,
		},
		{
			name: "zero timeout",
			env: map[string]string{
				"BACKEND_TIMEOUT": "0s",
			},
			wantErr: ErrInvalidTimeout,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Изолируем значения по умолчанию от окружения разработчика; Setenv восстановит его после теста.
			for _, key := range []string{
				"HTTP_ADDR",
				"GRPC_ADDR",
				"CATALOG_BACKEND_ADDRS",
				"ORDERS_BACKEND_ADDRS",
				"BACKEND_TIMEOUT",
				"AUTH_BACKEND_ADDRS",
				"AUTH_TIMEOUT",
			} {
				t.Setenv(key, "")
				assert.NoError(t, os.Unsetenv(key))
			}
			for key, value := range tt.env {
				t.Setenv(key, value)
			}
			got, err := Load()
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
		})
	}
}
