package upstream

import (
	"context"
	"testing"
	"time"

	ordersv1 "coursework/gateway/gen/orders/v1"
	"coursework/gateway/internal/gateway"
	"coursework/gateway/internal/upstream/mocks"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

func TestOrdersGetOrder(t *testing.T) {
	type getOrderMock struct {
		request *ordersv1.GetOrderRequest
		order   *ordersv1.Order
		err     error
	}

	tests := []struct {
		name           string
		id             string
		clientGetOrder *getOrderMock
		want           gateway.Order
		wantErr        error
	}{
		{
			name: "success",
			id:   "42",
			clientGetOrder: &getOrderMock{
				request: &ordersv1.GetOrderRequest{
					Id: "42",
				},
				order: &ordersv1.Order{
					Id:     "42",
					ItemId: "7",
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
			name: "error discards response",
			id:   "42",
			clientGetOrder: &getOrderMock{
				request: &ordersv1.GetOrderRequest{
					Id: "42",
				},
				order: &ordersv1.Order{
					Id:     "42",
					ItemId: "7",
					Status: "created",
				},
				err: assert.AnError,
			},
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := metadata.NewOutgoingContext(t.Context(), metadata.Pairs("x-request-id", "request-42"))
			var callCtx context.Context
			client := mocks.NewOrderServiceClientMock(t)
			if call := tt.clientGetOrder; call != nil {
				client.GetOrderMock.Expect(minimock.AnyContext, call.request).Return(call.order, call.err)
				client.GetOrderMock.Times(1).Inspect(func(ctx context.Context, _ *ordersv1.GetOrderRequest, _ ...grpc.CallOption) {
					callCtx = ctx
					md, _ := metadata.FromOutgoingContext(ctx)
					assert.Equal(t, []string{
						"request-42",
					}, md.Get("x-request-id"))
				})
			}

			orders := &Orders{
				client:  client,
				timeout: time.Hour,
			}
			got, err := orders.GetOrder(ctx, tt.id)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
			if assert.NotNil(t, callCtx, "backend was not called") {
				assert.ErrorIs(t, callCtx.Err(), context.Canceled)
			}
		})
	}
}

func TestOrdersGetOrderDeadline(t *testing.T) {
	type getOrderMock struct {
		request *ordersv1.GetOrderRequest
		order   *ordersv1.Order
		err     error
	}

	tests := []struct {
		name           string
		parentTimeout  time.Duration
		backendTimeout time.Duration
		clientGetOrder *getOrderMock
		wantParent     bool
	}{
		{
			name:           "backend timeout limits call",
			parentTimeout:  2 * time.Hour,
			backendTimeout: time.Hour,
			clientGetOrder: &getOrderMock{
				request: &ordersv1.GetOrderRequest{
					Id: "42",
				},
				order: &ordersv1.Order{},
			},
		},
		{
			name:           "earlier parent deadline is preserved",
			parentTimeout:  time.Hour,
			backendTimeout: 2 * time.Hour,
			clientGetOrder: &getOrderMock{
				request: &ordersv1.GetOrderRequest{
					Id: "42",
				},
				order: &ordersv1.Order{},
			},
			wantParent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(t.Context(), tt.parentTimeout)
			defer cancel()
			parentDeadline, _ := ctx.Deadline()
			var callCtx context.Context
			var calledAt time.Time
			client := mocks.NewOrderServiceClientMock(t)
			if call := tt.clientGetOrder; call != nil {
				client.GetOrderMock.Expect(minimock.AnyContext, call.request).Return(call.order, call.err)
				client.GetOrderMock.Times(1).Inspect(func(ctx context.Context, _ *ordersv1.GetOrderRequest, _ ...grpc.CallOption) {
					callCtx = ctx
					calledAt = time.Now()
				})
			}

			orders := &Orders{
				client:  client,
				timeout: tt.backendTimeout,
			}
			before := time.Now()
			_, err := orders.GetOrder(ctx, "42")
			assert.NoError(t, err)
			assert.NoError(t, ctx.Err(), "parent context must remain active")
			if !assert.NotNil(t, callCtx) {
				return
			}
			deadline, ok := callCtx.Deadline()
			if !assert.True(t, ok, "backend context must have a deadline") {
				return
			}
			if tt.wantParent {
				assert.Equal(t, parentDeadline, deadline)
			} else {
				assert.False(t, deadline.Before(before.Add(tt.backendTimeout)), "backend deadline is too early")
				assert.False(t, deadline.After(calledAt.Add(tt.backendTimeout)), "backend deadline is too late")
			}
		})
	}
}

func TestOrdersGetOrderCancellation(t *testing.T) {
	type getOrderMock struct {
		request *ordersv1.GetOrderRequest
		order   *ordersv1.Order
		err     error
	}

	tests := []struct {
		name           string
		backendTimeout time.Duration
		clientGetOrder *getOrderMock
		wantErr        error
	}{
		{
			name:           "parent canceled during call",
			backendTimeout: time.Hour,
			clientGetOrder: &getOrderMock{
				request: &ordersv1.GetOrderRequest{
					Id: "42",
				},
				err: context.Canceled,
			},
			wantErr: context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			client := mocks.NewOrderServiceClientMock(t)
			if call := tt.clientGetOrder; call != nil {
				client.GetOrderMock.Expect(minimock.AnyContext, call.request).Return(call.order, call.err)
				client.GetOrderMock.Times(1).Inspect(func(ctx context.Context, _ *ordersv1.GetOrderRequest, _ ...grpc.CallOption) {
					cancel()
					assert.ErrorIs(t, ctx.Err(), tt.wantErr)
				})
			}

			orders := &Orders{
				client:  client,
				timeout: tt.backendTimeout,
			}
			got, err := orders.GetOrder(ctx, "42")
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Empty(t, got)
		})
	}
}

func TestNewOrders(t *testing.T) {
	type getOrderMock struct {
		request *ordersv1.GetOrderRequest
		order   *ordersv1.Order
		err     error
	}

	tests := []struct {
		name           string
		id             string
		clientGetOrder *getOrderMock
		want           gateway.Order
		wantErr        error
	}{
		{
			name: "connection is used for RPC",
			id:   "42",
			clientGetOrder: &getOrderMock{
				request: &ordersv1.GetOrderRequest{
					Id: "42",
				},
				order: &ordersv1.Order{
					Id:     "42",
					ItemId: "7",
					Status: "created",
				},
			},
			want: gateway.Order{
				ID:     "42",
				ItemID: "7",
				Status: "created",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := mocks.NewOrderServiceClientMock(t)
			if call := tt.clientGetOrder; call != nil {
				client.GetOrderMock.Expect(minimock.AnyContext, call.request).Return(call.order, call.err)
				client.GetOrderMock.Times(1)
			}
			// Перехватчик заменяет вызов RPC; подключение к сети и запуск сервера не нужны.
			conn, err := grpc.NewClient("passthrough:///unused",
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithUnaryInterceptor(func(ctx context.Context, method string, request, reply any, _ *grpc.ClientConn, _ grpc.UnaryInvoker, _ ...grpc.CallOption) error {
					assert.Equal(t, ordersv1.OrderService_GetOrder_FullMethodName, method)
					_, ok := ctx.Deadline()
					assert.True(t, ok, "backend context must have a deadline")
					order, err := client.GetOrder(ctx, request.(*ordersv1.GetOrderRequest))
					if err == nil {
						proto.Merge(reply.(*ordersv1.Order), order)
					}
					return err
				}),
			)
			assert.NoError(t, err)
			t.Cleanup(func() { assert.NoError(t, conn.Close()) })

			got, err := NewOrders(conn, time.Hour).GetOrder(t.Context(), tt.id)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
		})
	}
}
