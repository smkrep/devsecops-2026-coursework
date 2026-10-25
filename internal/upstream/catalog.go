package upstream

import (
	"context"
	"time"

	catalogv1 "coursework/gateway/gen/catalog/v1"
	"coursework/gateway/internal/gateway"

	"google.golang.org/grpc"
)

type Catalog struct {
	client  catalogv1.CatalogServiceClient
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

	item, err := c.client.GetItem(ctx, &catalogv1.GetItemRequest{Id: id})
	if err != nil {
		return gateway.Item{}, err
	}
	return gateway.Item{ID: item.GetId(), Name: item.GetName()}, nil
}
