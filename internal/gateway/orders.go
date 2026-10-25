package gateway

import "context"

type Order struct {
	ID     string
	ItemID string
	Status string
}

// OrderBackend описывает обращения шлюза к сервису заказов.
//
//go:generate go tool minimock -i OrderBackend -o mocks/order_backend_mock.go -p mocks -g
type OrderBackend interface {
	GetOrder(ctx context.Context, id string) (Order, error)
}

type OrderService struct {
	backend OrderBackend
}

func NewOrderService(backend OrderBackend) *OrderService {
	return &OrderService{
		backend: backend,
	}
}

func (s *OrderService) GetOrder(ctx context.Context, id string) (Order, error) {
	return s.backend.GetOrder(ctx, id)
}
