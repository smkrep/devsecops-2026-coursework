package upstream

import (
	"context"
	"time"

	catalogv1 "github.com/smkrep/devsecops-2026-coursework/gen/catalog/v1"
	"github.com/smkrep/devsecops-2026-coursework/internal/gateway"

	"google.golang.org/grpc"
)

// CatalogClient описывает RPC каталога, используемый исходящим адаптером.
//
//go:generate go tool minimock -i CatalogClient -o mocks/catalog_client_mock.go -p mocks -g
type CatalogClient interface {
	GetItem(ctx context.Context, request *catalogv1.GetItemRequest, opts ...grpc.CallOption) (*catalogv1.Item, error)
}

type Catalog struct {
	client  CatalogClient
	timeout time.Duration
}

func NewCatalog(conn grpc.ClientConnInterface, timeout time.Duration) *Catalog {
	return &Catalog{
		client:  catalogv1.NewCatalogServiceClient(conn),
		timeout: timeout,
	}
}

func (c *Catalog) GetItem(ctx context.Context, id string) (gateway.Item, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	item, err := c.client.GetItem(ctx, &catalogv1.GetItemRequest{
		Id: id,
	})
	if err != nil {
		return gateway.Item{}, err
	}
	return gateway.Item{
		ID:   item.GetId(),
		Name: item.GetName(),
	}, nil
}
