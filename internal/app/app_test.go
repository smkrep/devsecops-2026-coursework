package app

import (
	"net"
	"testing"
	"time"

	"coursework/gateway/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestRunInvalidGRPCAddress(t *testing.T) {
	tests := []struct {
		name string
		addr string
	}{
		{
			name: "missing port",
			addr: "127.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Invalid numeric addresses fail before a socket or server is opened.
			cfg := config.Config{
				HTTPAddr:       "127.0.0.1:8080",
				GRPCAddr:       tt.addr,
				BackendAddr:    "127.0.0.1:9092",
				BackendTimeout: time.Second,
			}
			err := Run(t.Context(), cfg)
			assert.ErrorContains(t, err, "listen for gRPC:")
			var addrErr *net.AddrError
			assert.ErrorAs(t, err, &addrErr)
		})
	}
}
