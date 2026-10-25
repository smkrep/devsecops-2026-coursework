package grpctransport

import (
	"context"

	catalogv1 "coursework/gateway/gen/catalog/v1"
	"coursework/gateway/internal/gateway"
)

type Handler struct {
	catalogv1.UnimplementedCatalogServiceServer
	service *gateway.Service
}

func NewHandler(service *gateway.Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) GetItem(ctx context.Context, request *catalogv1.GetItemRequest) (*catalogv1.Item, error) {
	item, err := h.service.GetItem(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	return &catalogv1.Item{
		Id:   item.ID,
		Name: item.Name,
	}, nil
}
