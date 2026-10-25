package grpctransport

import (
	"context"
	"testing"

	ordersv1 "github.com/smkrep/devsecops-2026-coursework/gen/orders/v1"
	"github.com/smkrep/devsecops-2026-coursework/internal/gateway"
	"github.com/smkrep/devsecops-2026-coursework/internal/gateway/mocks"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestOrderHandlerGetOrder(t *testing.T) {
	type getOrderMock struct {
		id    string
		order gateway.Order
		err   error
	}

	tests := []struct {
		name            string
		request         *ordersv1.GetOrderRequest
		backendGetOrder *getOrderMock
		want            *ordersv1.Order
		wantErr         error
	}{
		{
			name: "success",
			request: &ordersv1.GetOrderRequest{
				Id: "42",
			},
			backendGetOrder: &getOrderMock{
				id: "42",
				order: gateway.Order{
					ID:     "42",
					ItemID: "7",
					Status: "created",
				},
			},
			want: &ordersv1.Order{
				Id:     "42",
				ItemId: "7",
				Status: "created",
			},
		},
		{
			name: "unexpected error discards order",
			request: &ordersv1.GetOrderRequest{
				Id: "42",
			},
			backendGetOrder: &getOrderMock{
				id: "42",
				order: gateway.Order{
					ID: "42",
				},
				err: assert.AnError,
			},
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			type contextKey struct{}
			ctx := context.WithValue(t.Context(), contextKey{}, "request-42")
			backend := mocks.NewOrderBackendMock(t)
			if call := tt.backendGetOrder; call != nil {
				backend.GetOrderMock.Expect(ctx, call.id).Return(call.order, call.err)
			}

			handler := NewOrderHandler(gateway.NewOrderService(backend))
			got, err := handler.GetOrder(ctx, tt.request)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.True(t, proto.Equal(tt.want, got), "GetOrder() = %v, want %v", got, tt.want)
		})
	}
}
