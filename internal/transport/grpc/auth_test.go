package grpctransport

import (
	"context"
	"fmt"
	"testing"

	"github.com/smkrep/devsecops-2026-coursework/internal/auth"
	"github.com/smkrep/devsecops-2026-coursework/internal/transport/grpc/mocks"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuthentication(t *testing.T) {
	type verifyMock struct {
		token    string
		identity auth.Identity
		err      error
	}

	tests := []struct {
		name           string
		authorization  []string
		verifierVerify *verifyMock
		handlerErr     error
		wantIdentity   auth.Identity
		wantCalled     bool
		want           any
		wantErr        error
	}{
		{
			name: "verified user replaces caller identity",
			authorization: []string{
				"Bearer token-42",
			},
			verifierVerify: &verifyMock{
				token: "token-42",
				identity: auth.Identity{
					UserID: "user-42",
				},
			},
			wantIdentity: auth.Identity{
				UserID: "user-42",
			},
			wantCalled: true,
			want:       "response",
		},
		{
			name: "bearer scheme ignores case and handler error is preserved",
			authorization: []string{
				"bearer token-42",
			},
			verifierVerify: &verifyMock{
				token: "token-42",
				identity: auth.Identity{
					UserID: "user-42",
				},
			},
			handlerErr: assert.AnError,
			wantIdentity: auth.Identity{
				UserID: "user-42",
			},
			wantCalled: true,
			want:       "response",
			wantErr:    assert.AnError,
		},
		{
			name:    "missing token",
			wantErr: status.Error(codes.Unauthenticated, "bearer token is required"),
		},
		{
			name: "unsupported scheme",
			authorization: []string{
				"Basic token-42",
			},
			wantErr: status.Error(codes.Unauthenticated, "bearer token is required"),
		},
		{
			name: "ambiguous authorization",
			authorization: []string{
				"Bearer token-42",
				"Bearer another-token",
			},
			wantErr: status.Error(codes.Unauthenticated, "bearer token is required"),
		},
		{
			name: "invalid token hides auth service details",
			authorization: []string{
				"Bearer token-42",
			},
			verifierVerify: &verifyMock{
				token: "token-42",
				err:   status.Error(codes.Unauthenticated, "expired token-42"),
			},
			wantErr: status.Error(codes.Unauthenticated, "invalid bearer token"),
		},
		{
			name: "auth service unavailable",
			authorization: []string{
				"Bearer token-42",
			},
			verifierVerify: &verifyMock{
				token: "token-42",
				err:   assert.AnError,
			},
			wantErr: status.Error(codes.Unavailable, "authentication service unavailable"),
		},
		{
			name: "invalid identity cannot reach backend",
			authorization: []string{
				"Bearer token-42",
			},
			verifierVerify: &verifyMock{
				token: "token-42",
				identity: auth.Identity{
					UserID: "user\n42",
				},
			},
			wantErr: status.Error(codes.Unavailable, "authentication service unavailable"),
		},
		{
			name: "parent cancellation hides wrapped details",
			authorization: []string{
				"Bearer token-42",
			},
			verifierVerify: &verifyMock{
				token: "token-42",
				err:   fmt.Errorf("token-42: %w", context.Canceled),
			},
			wantErr: status.FromContextError(context.Canceled).Err(),
		},
		{
			name: "auth deadline exceeded",
			authorization: []string{
				"Bearer token-42",
			},
			verifierVerify: &verifyMock{
				token: "token-42",
				err:   status.Error(codes.DeadlineExceeded, "private deadline details"),
			},
			wantErr: status.Error(codes.DeadlineExceeded, "authentication deadline exceeded"),
		},
		{
			name: "access denied",
			authorization: []string{
				"Bearer token-42",
			},
			verifierVerify: &verifyMock{
				token: "token-42",
				err:   status.Error(codes.PermissionDenied, "private policy details"),
			},
			wantErr: status.Error(codes.PermissionDenied, "access denied"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			incoming := metadata.Pairs("x-user-id", "forged-user")
			incoming.Set("authorization", tt.authorization...)
			ctx := metadata.NewIncomingContext(t.Context(), incoming)
			ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer must-not-leak", "grpcgateway-authorization", "Bearer must-not-leak", "x-user-id", "forged-user", "x-request-id", "request-42"))
			verifier := mocks.NewVerifierMock(t)
			if call := tt.verifierVerify; call != nil {
				verifier.VerifyMock.Expect(ctx, call.token).Return(call.identity, call.err)
				verifier.VerifyMock.Times(1)
			}
			called := false
			handler := func(ctx context.Context, request any) (any, error) {
				called = true
				assert.Equal(t, "request", request)
				identity, ok := auth.IdentityFromContext(ctx)
				assert.True(t, ok)
				assert.Equal(t, tt.wantIdentity, identity)
				outgoing, _ := metadata.FromOutgoingContext(ctx)
				assert.Equal(t, []string{
					tt.wantIdentity.UserID,
				}, outgoing.Get("x-user-id"))
				assert.Empty(t, outgoing.Get("authorization"))
				assert.Empty(t, outgoing.Get("grpcgateway-authorization"))
				assert.Equal(t, []string{
					"request-42",
				}, outgoing.Get("x-request-id"))
				return "response", tt.handlerErr
			}
			got, err := Authentication(verifier)(ctx, "request", &grpc.UnaryServerInfo{}, handler)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantCalled, called)
			assert.Equal(t, []string{
				"forged-user",
			}, incoming.Get("x-user-id"))
		})
	}
}
