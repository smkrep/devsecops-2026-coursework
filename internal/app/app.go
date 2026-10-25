package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	catalogv1 "github.com/smkrep/devsecops-2026-coursework/gen/catalog/v1"
	ordersv1 "github.com/smkrep/devsecops-2026-coursework/gen/orders/v1"
	"github.com/smkrep/devsecops-2026-coursework/internal/config"
	"github.com/smkrep/devsecops-2026-coursework/internal/gateway"
	grpctransport "github.com/smkrep/devsecops-2026-coursework/internal/transport/grpc"
	httptransport "github.com/smkrep/devsecops-2026-coursework/internal/transport/http"
	"github.com/smkrep/devsecops-2026-coursework/internal/upstream"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Run(ctx context.Context, cfg config.Config) error {
	// NewClient устанавливает соединение лениво: шлюз запускается даже при недоступном backend.
	backendConn, err := upstream.NewBackendConn(cfg.CatalogBackendAddrs)
	if err != nil {
		return fmt.Errorf("create catalog backend client: %w", err)
	}
	defer backendConn.Close()

	ordersConn, err := upstream.NewBackendConn(cfg.OrdersBackendAddrs)
	if err != nil {
		return fmt.Errorf("create orders backend client: %w", err)
	}
	defer ordersConn.Close()

	authConn, err := upstream.NewBackendConn(cfg.AuthBackendAddrs)
	if err != nil {
		return fmt.Errorf("create authentication client: %w", err)
	}
	defer authConn.Close()

	rateLimitConn, err := upstream.NewBackendConn(cfg.RateLimitBackendAddrs)
	if err != nil {
		return fmt.Errorf("create rate limit client: %w", err)
	}
	defer rateLimitConn.Close()

	grpcListener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen for gRPC: %w", err)
	}
	defer grpcListener.Close()

	verifier := upstream.NewAuthentication(authConn, cfg.AuthTimeout)
	limiter := upstream.NewRateLimit(rateLimitConn, cfg.RateLimitTimeout)
	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(
		grpctransport.Authentication(verifier),
		grpctransport.RateLimiting(limiter),
	))
	service := gateway.NewService(upstream.NewCatalog(backendConn, cfg.BackendTimeout))
	catalogv1.RegisterCatalogServiceServer(grpcServer, grpctransport.NewHandler(service))
	orders := gateway.NewOrderService(upstream.NewOrders(ordersConn, cfg.BackendTimeout))
	ordersv1.RegisterOrderServiceServer(grpcServer, grpctransport.NewOrderHandler(orders))

	// Подключаемся через loopback, даже если gRPC-сервер слушает на всех интерфейсах.
	localAddr := grpcListener.Addr().(*net.TCPAddr)
	localIP := localAddr.IP
	if localIP.IsUnspecified() {
		if localIP.To4() != nil {
			localIP = net.IPv4(127, 0, 0, 1)
		} else {
			localIP = net.IPv6loopback
		}
	}
	localTarget := (&net.TCPAddr{
		IP:   localIP,
		Port: localAddr.Port,
	}).String()
	localConn, err := grpc.NewClient(localTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("create local gRPC client: %w", err)
	}
	defer localConn.Close()

	httpServer, err := httptransport.NewServer(ctx, localConn)
	if err != nil {
		return fmt.Errorf("create HTTP adapter: %w", err)
	}
	httpListener, err := net.Listen("tcp", cfg.HTTPAddr)
	if err != nil {
		return fmt.Errorf("listen for HTTP: %w", err)
	}
	defer httpListener.Close()

	serveErrors := make(chan error, 2)
	go func() {
		serveErrors <- grpcServer.Serve(grpcListener)
	}()
	go func() {
		serveErrors <- httpServer.Serve(httpListener)
	}()
	slog.Info("gateway started", "http", httpListener.Addr(), "grpc", grpcListener.Addr(), "catalog_backends", cfg.CatalogBackendAddrs, "orders_backends", cfg.OrdersBackendAddrs)

	var serveErr error
	select {
	case <-ctx.Done():
	case serveErr = <-serveErrors:
	}

	// Завершаем HTTP-запросы перед остановкой локального gRPC-сервера. Ограничиваем общее время остановки.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	shutdownErr := httpServer.Shutdown(shutdownCtx)
	if shutdownErr != nil {
		_ = httpServer.Close()
	}
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-shutdownCtx.Done():
		grpcServer.Stop()
		<-stopped
	}
	if errors.Is(serveErr, http.ErrServerClosed) || errors.Is(serveErr, grpc.ErrServerStopped) {
		serveErr = nil
	}
	return errors.Join(serveErr, shutdownErr)
}
