package grpctransport

import (
	"context"
	"errors"

	"github.com/smkrep/devsecops-2026-coursework/internal/auth"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RateLimiter описывает проверку лимита для пользователя и ресурса.
//
//go:generate go tool minimock -i RateLimiter -o mocks/rate_limiter_mock.go -p mocks -g
type RateLimiter interface {
	Allow(ctx context.Context, userID string, resource string) (bool, error)
}

// RateLimiting проверяет лимит после аутентификации, до вызова backend.
func RateLimiting(limiter RateLimiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, request any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		identity, ok := auth.IdentityFromContext(ctx)
		if !ok || !auth.ValidIdentity(identity) {
			return nil, status.Error(codes.Unauthenticated, "authenticated user is required")
		}
		allowed, err := limiter.Allow(ctx, identity.UserID, info.FullMethod)
		if err != nil {
			return nil, rateLimitError(err)
		}
		if !allowed {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}
		return handler(ctx, request)
	}
}

func rateLimitError(err error) error {
	if errors.Is(err, context.Canceled) {
		return status.FromContextError(context.Canceled).Err()
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return status.FromContextError(context.DeadlineExceeded).Err()
	}
	switch status.Code(err) {
	case codes.Canceled:
		return status.Error(codes.Canceled, "rate limit check canceled")
	case codes.DeadlineExceeded:
		return status.Error(codes.DeadlineExceeded, "rate limit deadline exceeded")
	default:
		// Ошибка проверки не разрешает запрос и не раскрывает внутренние сведения сервиса.
		return status.Error(codes.Unavailable, "rate limit service unavailable")
	}
}
