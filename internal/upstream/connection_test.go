package upstream

import (
	"testing"

	"coursework/gateway/internal/upstream/mocks"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/serviceconfig"
)

func TestNewBackendConn(t *testing.T) {
	tests := []struct {
		name      string
		addresses []string
		wantErr   error
		wantConn  bool
	}{
		{
			name: "single backend",
			addresses: []string{
				"catalog:9092",
			},
			wantConn: true,
		},
		{
			name: "multiple backends",
			addresses: []string{
				"orders:9093",
				"orders:9095",
			},
			wantConn: true,
		},
		{
			name:    "no backends",
			wantErr: ErrNoBackendAddresses,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conn, err := NewBackendConn(tt.addresses)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantConn, conn != nil)
			if conn != nil {
				t.Cleanup(func() { assert.NoError(t, conn.Close()) })
				assert.Equal(t, "backend:///backend", conn.CanonicalTarget())
				assert.Equal(t, connectivity.Idle, conn.GetState(), "создание клиента не должно подключать его к backend")
			}
		})
	}
}

func TestBackendResolverBuild(t *testing.T) {
	type parseServiceConfigMock struct {
		config string
		result *serviceconfig.ParseResult
	}
	type updateStateMock struct {
		state resolver.State
		err   error
	}

	parsed := &serviceconfig.ParseResult{}
	wantConfig := `{"loadBalancingConfig":[{"round_robin":{}}]}`
	catalogState := resolver.State{
		ServiceConfig: parsed,
		Endpoints: []resolver.Endpoint{
			{
				Addresses: []resolver.Address{
					{
						Addr: "catalog:9092",
					},
				},
			},
		},
	}
	ordersState := resolver.State{
		ServiceConfig: parsed,
		Endpoints: []resolver.Endpoint{
			{
				Addresses: []resolver.Address{
					{
						Addr: "orders:9093",
					},
				},
			},
			{
				Addresses: []resolver.Address{
					{
						Addr: "[::1]:9095",
					},
				},
			},
		},
	}
	tests := []struct {
		name               string
		addresses          []string
		parseServiceConfig *parseServiceConfigMock
		updateState        *updateStateMock
		wantErr            error
		wantResolver       bool
	}{
		{
			name:         "single catalog endpoint with round robin",
			wantResolver: true,
			addresses: []string{
				"catalog:9092",
			},
			parseServiceConfig: &parseServiceConfigMock{
				config: wantConfig,
				result: parsed,
			},
			updateState: &updateStateMock{
				state: catalogState,
			},
		},
		{
			name:         "orders instances are separate endpoints",
			wantResolver: true,
			addresses: []string{
				"orders:9093",
				"[::1]:9095",
			},
			parseServiceConfig: &parseServiceConfigMock{
				config: wantConfig,
				result: parsed,
			},
			updateState: &updateStateMock{
				state: ordersState,
			},
		},
		{
			name: "service config error",
			addresses: []string{
				"catalog:9092",
			},
			parseServiceConfig: &parseServiceConfigMock{
				config: wantConfig,
				result: &serviceconfig.ParseResult{
					Err: assert.AnError,
				},
			},
			wantErr: assert.AnError,
		},
		{
			name: "endpoint update error",
			addresses: []string{
				"catalog:9092",
			},
			parseServiceConfig: &parseServiceConfigMock{
				config: wantConfig,
				result: parsed,
			},
			updateState: &updateStateMock{
				state: catalogState,
				err:   assert.AnError,
			},
			wantErr: assert.AnError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cc := mocks.NewClientConnMock(t)
			if call := tt.parseServiceConfig; call != nil {
				cc.ParseServiceConfigMock.Expect(call.config).Return(call.result)
				cc.ParseServiceConfigMock.Times(1)
			}
			if call := tt.updateState; call != nil {
				cc.UpdateStateMock.Expect(call.state).Return(call.err)
				cc.UpdateStateMock.Times(1)
			}
			builder := &backendResolver{
				addresses: tt.addresses,
			}
			got, err := builder.Build(resolver.Target{}, cc, resolver.BuildOptions{})
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantResolver, got != nil)
			if got != nil {
				assert.Same(t, builder, got)
				// Фиксированный список не требует повторного разрешения адресов и обновления соединения.
				got.ResolveNow(resolver.ResolveNowOptions{})
				got.Close()
			}
		})
	}
}
