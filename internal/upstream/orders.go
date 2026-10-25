package upstream

import (
	"context"
	"time"

	ordersv1 "coursework/gateway/gen/orders/v1"
	"coursework/gateway/internal/gateway"

	"google.golang.org/grpc"
)

type Orders struct {
	client  ordersv1.OrderServiceClient
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
