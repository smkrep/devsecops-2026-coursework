package httptransport

import (
	"context"
	"net/http"
	"time"

	catalogv1 "coursework/gateway/gen/catalog/v1"
	ordersv1 "coursework/gateway/gen/orders/v1"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

func NewServer(ctx context.Context, conn *grpc.ClientConn) (*http.Server, error) {
	mux := runtime.NewServeMux()
	if err := catalogv1.RegisterCatalogServiceHandler(ctx, mux, conn); err != nil {
		return nil, err
	}
	if err := ordersv1.RegisterOrderServiceHandler(ctx, mux, conn); err != nil {
		return nil, err
	}
	return &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}, nil
}
