package upstream

import (
	"context"
	"time"

	ordersv1 "github.com/smkrep/devsecops-2026-coursework/gen/orders/v1"
	"github.com/smkrep/devsecops-2026-coursework/internal/gateway"

	"google.golang.org/grpc"
)

// OrderClient описывает RPC заказов, используемый исходящим адаптером.
//
//go:generate go tool minimock -i OrderClient -o mocks/order_client_mock.go -p mocks -g
type OrderClient interface {
	GetOrder(ctx context.Context, request *ordersv1.GetOrderRequest, opts ...grpc.CallOption) (*ordersv1.Order, error)
}

type Orders struct {
	client  OrderClient
	timeout time.Duration
}

func NewOrders(conn grpc.ClientConnInterface, timeout time.Duration) *Orders {
	return &Orders{
		client:  ordersv1.NewOrderServiceClient(conn),
		timeout: timeout,
	}
}

func (c *Orders) GetOrder(ctx context.Context, id string) (gateway.Order, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	order, err := c.client.GetOrder(ctx, &ordersv1.GetOrderRequest{
		Id: id,
	})
	if err != nil {
		return gateway.Order{}, err
	}
	return gateway.Order{
		ID:     order.GetId(),
		ItemID: order.GetItemId(),
		Status: order.GetStatus(),
	}, nil
}
