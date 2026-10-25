package app

import (
	"net"
	"testing"
	"time"

	"github.com/smkrep/devsecops-2026-coursework/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestRunInvalidGRPCAddress(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		wantErr *net.AddrError
	}{
		{
			name: "missing port",
			addr: "127.0.0.1",
			wantErr: &net.AddrError{
				Err:  "missing port in address",
				Addr: "127.0.0.1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Некорректный адрес с числовым IP вызывает ошибку до открытия сокета и запуска сервера.
			cfg := config.Config{
				HTTPAddr: "127.0.0.1:8080",
				GRPCAddr: tt.addr,
				CatalogBackendAddrs: []string{
					"127.0.0.1:9092",
				},
				OrdersBackendAddrs: []string{
					"127.0.0.1:9093",
				},
				BackendTimeout: time.Second,
				AuthBackendAddrs: []string{
					"127.0.0.1:9096",
				},
				AuthTimeout: time.Second,
			}
			err := Run(t.Context(), cfg)
			var addrErr *net.AddrError
			assert.ErrorAs(t, err, &addrErr)
			assert.Equal(t, tt.wantErr, addrErr)
		})
	}
}
