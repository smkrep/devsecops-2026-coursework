package grpctransport

import (
	"context"
	"errors"
	"strings"

	"github.com/smkrep/devsecops-2026-coursework/internal/auth"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Verifier описывает проверку токена, необходимую перехватчику аутентификации.
//
//go:generate go tool minimock -i Verifier -o mocks/verifier_mock.go -p mocks -g
type Verifier interface {
	Verify(ctx context.Context, token string) (auth.Identity, error)
}

// Authentication проверяет токен до вызова обработчика HTTP или прямого gRPC-входа.
func Authentication(verifier Verifier) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, request any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, _ := metadata.FromIncomingContext(ctx)
		values := md.Get("authorization")
		if len(values) != 1 {
			return nil, status.Error(codes.Unauthenticated, "bearer token is required")
		}
		parts := strings.Fields(values[0])
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return nil, status.Error(codes.Unauthenticated, "bearer token is required")
		}
		identity, err := verifier.Verify(ctx, parts[1])
		if err != nil {
			return nil, authenticationError(err)
		}
		if !auth.ValidIdentity(identity) {
			return nil, status.Error(codes.Unavailable, "authentication service unavailable")
		}

		ctx = auth.WithIdentity(ctx, identity)
		// Доверяем только проверенному ответу; входящие заголовки пользователя не копируем.
		outgoing, _ := metadata.FromOutgoingContext(ctx)
		outgoing = outgoing.Copy()
		outgoing.Delete("authorization")
		outgoing.Delete("grpcgateway-authorization")
		outgoing.Set("x-user-id", identity.UserID)
		ctx = metadata.NewOutgoingContext(ctx, outgoing)
		return handler(ctx, request)
	}
}

func authenticationError(err error) error {
	if errors.Is(err, context.Canceled) {
		return status.FromContextError(context.Canceled).Err()
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return status.FromContextError(context.DeadlineExceeded).Err()
	}
	switch status.Code(err) {
	case codes.Unauthenticated:
		return status.Error(codes.Unauthenticated, "invalid bearer token")
	case codes.PermissionDenied:
		return status.Error(codes.PermissionDenied, "access denied")
	case codes.Canceled:
		return status.Error(codes.Canceled, "authentication canceled")
	case codes.DeadlineExceeded:
		return status.Error(codes.DeadlineExceeded, "authentication deadline exceeded")
	default:
		// Не раскрываем токен и внутренние сведения из ошибки внешнего сервиса.
		return status.Error(codes.Unavailable, "authentication service unavailable")
	}
}
