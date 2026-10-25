package gateway_test

import (
	"testing"

	"github.com/smkrep/devsecops-2026-coursework/internal/gateway"
	"github.com/smkrep/devsecops-2026-coursework/internal/gateway/mocks"

	"github.com/stretchr/testify/assert"
)

func TestOrderServiceGetOrder(t *testing.T) {
	type getOrderMock struct {
		id    string
		order gateway.Order
		err   error
	}

	tests := []struct {
		name            string
		id              string
		backendGetOrder *getOrderMock
		want            gateway.Order
		wantErr         error
	}{
		{
			name: "success",
			id:   "42",
			backendGetOrder: &getOrderMock{
				id: "42",
				order: gateway.Order{
					ID:     "42",
					ItemID: "7",
					Status: "created",
				},
			},
			want: gateway.Order{
				ID:     "42",
				ItemID: "7",
				Status: "created",
			},
		},
		{
			name: "backend error",
			id:   "42",
			backendGetOrder: &getOrderMock{
				id:  "42",
				err: assert.AnError,
			},
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			backend := mocks.NewOrderBackendMock(t)
			if call := tt.backendGetOrder; call != nil {
				backend.GetOrderMock.Expect(ctx, call.id).Return(call.order, call.err)
			}

			got, err := gateway.NewOrderService(backend).GetOrder(ctx, tt.id)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
		})
	}
}
