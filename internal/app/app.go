package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	catalogv1 "coursework/gateway/gen/catalog/v1"
	"coursework/gateway/internal/config"
	"coursework/gateway/internal/gateway"
	grpctransport "coursework/gateway/internal/transport/grpc"
	httptransport "coursework/gateway/internal/transport/http"
	"coursework/gateway/internal/upstream"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Run(ctx context.Context, cfg config.Config) error {
	// NewClient connects lazily: the gateway can start while the backend is offline.
	backendConn, err := grpc.NewClient(cfg.BackendAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("create backend client: %w", err)
	}
	defer backendConn.Close()

	grpcListener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen for gRPC: %w", err)
	}
	defer grpcListener.Close()

	grpcServer := grpc.NewServer()
	service := gateway.NewService(upstream.NewCatalog(backendConn, cfg.BackendTimeout))
	catalogv1.RegisterCatalogServiceServer(grpcServer, grpctransport.NewHandler(service))

	// Dial loopback even when the gRPC listener is bound to a wildcard address.
	localAddr := grpcListener.Addr().(*net.TCPAddr)
	localIP := localAddr.IP
	if localIP.IsUnspecified() {
		if localIP.To4() != nil {
			localIP = net.IPv4(127, 0, 0, 1)
		} else {
			localIP = net.IPv6loopback
		}
	}
	localTarget := (&net.TCPAddr{IP: localIP, Port: localAddr.Port}).String()
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
	go func() { serveErrors <- grpcServer.Serve(grpcListener) }()
	go func() { serveErrors <- httpServer.Serve(httpListener) }()
	slog.Info("gateway started", "http", httpListener.Addr(), "grpc", grpcListener.Addr(), "backend", cfg.BackendAddr)

	var serveErr error
	select {
	case <-ctx.Done():
	case serveErr = <-serveErrors:
	}

	// Drain HTTP before stopping its local gRPC upstream. Bound the entire shutdown.
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
