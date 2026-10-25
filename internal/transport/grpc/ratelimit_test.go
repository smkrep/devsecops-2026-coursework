package grpctransport

import (
	"context"
	"fmt"
	"testing"

	catalogv1 "github.com/smkrep/devsecops-2026-coursework/gen/catalog/v1"
	ordersv1 "github.com/smkrep/devsecops-2026-coursework/gen/orders/v1"
	"github.com/smkrep/devsecops-2026-coursework/internal/auth"
	"github.com/smkrep/devsecops-2026-coursework/internal/transport/grpc/mocks"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestRateLimiting(t *testing.T) {
	type allowMock struct {
		userID   string
		resource string
		allowed  bool
		err      error
	}

	user := auth.Identity{
		UserID: "user-42",
	}
	tests := []struct {
		name         string
		identity     *auth.Identity
		method       string
		limiterAllow *allowMock
		handlerErr   error
		wantCalled   bool
		want         any
		wantErr      error
	}{
		{
			name:     "allowed catalog request uses verified identity",
			identity: &user,
			method:   catalogv1.CatalogService_GetItem_FullMethodName,
			limiterAllow: &allowMock{
				userID:   "user-42",
				resource: catalogv1.CatalogService_GetItem_FullMethodName,
				allowed:  true,
			},
			wantCalled: true,
			want:       "response",
		},
		{
			name:     "orders method and handler error are preserved",
			identity: &user,
			method:   ordersv1.OrderService_GetOrder_FullMethodName,
			limiterAllow: &allowMock{
				userID:   "user-42",
				resource: ordersv1.OrderService_GetOrder_FullMethodName,
				allowed:  true,
			},
			handlerErr: assert.AnError,
			wantCalled: true,
			want:       "response",
			wantErr:    assert.AnError,
		},
		{
			name:     "limit exceeded does not call backend",
			identity: &user,
			method:   catalogv1.CatalogService_GetItem_FullMethodName,
			limiterAllow: &allowMock{
				userID:   "user-42",
				resource: catalogv1.CatalogService_GetItem_FullMethodName,
			},
			wantErr: status.Error(codes.ResourceExhausted, "rate limit exceeded"),
		},
		{
			name:    "unverified identity does not reach limiter",
			method:  catalogv1.CatalogService_GetItem_FullMethodName,
			wantErr: status.Error(codes.Unauthenticated, "authenticated user is required"),
		},
		{
			name:     "service error cannot allow request or expose details",
			identity: &user,
			method:   catalogv1.CatalogService_GetItem_FullMethodName,
			limiterAllow: &allowMock{
				userID:   "user-42",
				resource: catalogv1.CatalogService_GetItem_FullMethodName,
				allowed:  true,
				err:      assert.AnError,
			},
			wantErr: status.Error(codes.Unavailable, "rate limit service unavailable"),
		},
		{
			name:     "parent cancellation hides wrapped details",
			identity: &user,
			method:   catalogv1.CatalogService_GetItem_FullMethodName,
			limiterAllow: &allowMock{
				userID:   "user-42",
				resource: catalogv1.CatalogService_GetItem_FullMethodName,
				err:      fmt.Errorf("private details: %w", context.Canceled),
			},
			wantErr: status.FromContextError(context.Canceled).Err(),
		},
		{
			name:     "service deadline exceeded",
			identity: &user,
			method:   catalogv1.CatalogService_GetItem_FullMethodName,
			limiterAllow: &allowMock{
				userID:   "user-42",
				resource: catalogv1.CatalogService_GetItem_FullMethodName,
				err:      status.Error(codes.DeadlineExceeded, "private deadline details"),
			},
			wantErr: status.Error(codes.DeadlineExceeded, "rate limit deadline exceeded"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs("x-user-id", "forged-user"))
			if tt.identity != nil {
				ctx = auth.WithIdentity(ctx, *tt.identity)
			}
			limiter := mocks.NewRateLimiterMock(t)
			if call := tt.limiterAllow; call != nil {
				limiter.AllowMock.Expect(ctx, call.userID, call.resource).Return(call.allowed, call.err)
				limiter.AllowMock.Times(1)
			}
			called := false
			handler := func(gotCtx context.Context, request any) (any, error) {
				called = true
				assert.Same(t, ctx, gotCtx)
				assert.Equal(t, "request", request)
				return "response", tt.handlerErr
			}
			info := &grpc.UnaryServerInfo{
				FullMethod: tt.method,
			}
			got, err := RateLimiting(limiter)(ctx, "request", info, handler)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantCalled, called)
		})
	}
}
