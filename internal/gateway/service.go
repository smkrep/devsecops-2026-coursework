package gateway

import "context"

type Item struct {
	ID   string
	Name string
}

// CatalogBackend is the outgoing dependency of the gateway.
type CatalogBackend interface {
	GetItem(ctx context.Context, id string) (Item, error)
}

type Service struct {
	backend CatalogBackend
}

func NewService(backend CatalogBackend) *Service {
	return &Service{backend: backend}
}

func (s *Service) GetItem(ctx context.Context, id string) (Item, error) {
	return s.backend.GetItem(ctx, id)
}
