package grpctransport

import (
	"context"

	ordersv1 "coursework/gateway/gen/orders/v1"
	"coursework/gateway/internal/gateway"
)

type OrderHandler struct {
	ordersv1.UnimplementedOrderServiceServer
	service *gateway.OrderService
}

func NewOrderHandler(service *gateway.OrderService) *OrderHandler {
	return &OrderHandler{
		service: service,
	}
}

func (h *OrderHandler) GetOrder(ctx context.Context, request *ordersv1.GetOrderRequest) (*ordersv1.Order, error) {
	order, err := h.service.GetOrder(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	return &ordersv1.Order{
		Id:     order.ID,
		ItemId: order.ItemID,
		Status: order.Status,
	}, nil
}
