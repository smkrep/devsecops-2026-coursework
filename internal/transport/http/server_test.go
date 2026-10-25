package httptransport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	catalogv1 "coursework/gateway/gen/catalog/v1"
	ordersv1 "coursework/gateway/gen/orders/v1"
	"coursework/gateway/internal/upstream/mocks"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func TestServerHandler(t *testing.T) {
	type getItemMock struct {
		request *catalogv1.GetItemRequest
		item    *catalogv1.Item
		err     error
	}

	type getOrderMock struct {
		request *ordersv1.GetOrderRequest
		order   *ordersv1.Order
		err     error
	}

	tests := []struct {
		name           string
		path           string
		clientGetItem  *getItemMock
		clientGetOrder *getOrderMock
		wantStatus     int
		wantCode       codes.Code
		wantItem       *catalogv1.Item
		wantOrder      *ordersv1.Order
		wantMessage    string
	}{
		{
			name: "success",
			path: "/api/v1/items/42",
			clientGetItem: &getItemMock{
				request: &catalogv1.GetItemRequest{
					Id: "42",
				},
				item: &catalogv1.Item{
					Id:   "42",
					Name: "Book",
				},
			},
			wantStatus: http.StatusOK,
			wantItem: &catalogv1.Item{
				Id:   "42",
				Name: "Book",
			},
		},
		{
			name: "not found",
			path: "/api/v1/items/missing",
			clientGetItem: &getItemMock{
				request: &catalogv1.GetItemRequest{
					Id: "missing",
				},
				err: status.Error(codes.NotFound, "item not found"),
			},
			wantStatus:  http.StatusNotFound,
			wantCode:    codes.NotFound,
			wantMessage: "item not found",
		},
		{
			name: "unavailable",
			path: "/api/v1/items/42",
			clientGetItem: &getItemMock{
				request: &catalogv1.GetItemRequest{
					Id: "42",
				},
				err: status.Error(codes.Unavailable, "backend offline"),
			},
			wantStatus:  http.StatusServiceUnavailable,
			wantCode:    codes.Unavailable,
			wantMessage: "backend offline",
		},
		{
			name: "deadline exceeded",
			path: "/api/v1/items/42",
			clientGetItem: &getItemMock{
				request: &catalogv1.GetItemRequest{
					Id: "42",
				},
				err: status.Error(codes.DeadlineExceeded, "backend timeout"),
			},
			wantStatus:  http.StatusGatewayTimeout,
			wantCode:    codes.DeadlineExceeded,
			wantMessage: "backend timeout",
		},
		{
			name: "orders route",
			path: "/api/v1/orders/42",
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
			wantStatus: http.StatusOK,
			wantOrder: &ordersv1.Order{
				Id:     "42",
				ItemId: "7",
				Status: "created",
			},
		},
		{
			name: "orders backend unavailable",
			path: "/api/v1/orders/42",
			clientGetOrder: &getOrderMock{
				request: &ordersv1.GetOrderRequest{
					Id: "42",
				},
				err: status.Error(codes.Unavailable, "orders offline"),
			},
			wantStatus:  http.StatusServiceUnavailable,
			wantCode:    codes.Unavailable,
			wantMessage: "orders offline",
		},
		{
			name:       "unknown route does not call backend",
			path:       "/missing",
			wantStatus: http.StatusNotFound,
			wantCode:   codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := mocks.NewCatalogServiceClientMock(t)
			if call := tt.clientGetItem; call != nil {
				client.GetItemMock.Expect(minimock.AnyContext, call.request).Return(call.item, call.err)
				client.GetItemMock.Times(1)
			}
			ordersClient := mocks.NewOrderServiceClientMock(t)
			if call := tt.clientGetOrder; call != nil {
				ordersClient.GetOrderMock.Expect(minimock.AnyContext, call.request).Return(call.order, call.err)
				ordersClient.GetOrderMock.Times(1)
			}
			// Подменяем только исходящий RPC; проверяем обработчик без запуска сетевого сервера.
			conn, err := grpc.NewClient("passthrough:///unused",
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithUnaryInterceptor(func(ctx context.Context, method string, request, reply any, _ *grpc.ClientConn, _ grpc.UnaryInvoker, _ ...grpc.CallOption) error {
					if method == ordersv1.OrderService_GetOrder_FullMethodName {
						order, err := ordersClient.GetOrder(ctx, request.(*ordersv1.GetOrderRequest))
						if err == nil {
							proto.Merge(reply.(*ordersv1.Order), order)
						}
						return err
					}
					assert.Equal(t, catalogv1.CatalogService_GetItem_FullMethodName, method)
					item, err := client.GetItem(ctx, request.(*catalogv1.GetItemRequest))
					if err == nil {
						proto.Merge(reply.(*catalogv1.Item), item)
					}
					return err
				}),
			)
			assert.NoError(t, err)
			t.Cleanup(func() { assert.NoError(t, conn.Close()) })

			server, err := NewServer(t.Context(), conn)
			assert.NoError(t, err)
			assert.Equal(t, 5*time.Second, server.ReadHeaderTimeout)
			assert.Equal(t, 60*time.Second, server.IdleTimeout)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, tt.path, nil).WithContext(t.Context())
			server.Handler.ServeHTTP(recorder, request)

			if !assert.Equal(t, tt.wantStatus, recorder.Code, "body: %s", recorder.Body) {
				return
			}
			assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
			if tt.wantOrder != nil {
				var order struct {
					ID     string `json:"id"`
					ItemID string `json:"itemId"`
					Status string `json:"status"`
				}
				assert.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &order))
				assert.Equal(t, tt.wantOrder.Id, order.ID)
				assert.Equal(t, tt.wantOrder.ItemId, order.ItemID)
				assert.Equal(t, tt.wantOrder.Status, order.Status)
				return
			}
			if tt.wantStatus == http.StatusOK {
				var item struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}
				assert.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &item))
				assert.Equal(t, tt.wantItem.Id, item.ID)
				assert.Equal(t, tt.wantItem.Name, item.Name)
				return
			}

			var body struct {
				Code    codes.Code `json:"code"`
				Message string     `json:"message"`
			}
			assert.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
			assert.Equal(t, tt.wantCode, body.Code)
			if tt.wantMessage != "" {
				assert.Equal(t, tt.wantMessage, body.Message)
			}
		})
	}
}
