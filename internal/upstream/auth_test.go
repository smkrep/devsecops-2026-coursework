package upstream

import (
	"context"
	"testing"
	"time"

	authv1 "github.com/smkrep/devsecops-2026-coursework/gen/auth/v1"
	"github.com/smkrep/devsecops-2026-coursework/internal/auth"
	"github.com/smkrep/devsecops-2026-coursework/internal/upstream/mocks"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestAuthenticationVerify(t *testing.T) {
	type validateTokenMock struct {
		request  *authv1.ValidateTokenRequest
		response *authv1.ValidateTokenResponse
		err      error
	}

	tests := []struct {
		name                string
		clientValidateToken *validateTokenMock
		parentTimeout       time.Duration
		cancelDuringCall    bool
		wantContextErr      error
		want                auth.Identity
		wantErr             error
	}{
		{
			name: "verified user",
			clientValidateToken: &validateTokenMock{
				request: &authv1.ValidateTokenRequest{
					Token: "token-42",
				},
				response: &authv1.ValidateTokenResponse{
					UserId: "user-42",
				},
			},
			want: auth.Identity{
				UserID: "user-42",
			},
		},
		{
			name: "error discards identity",
			clientValidateToken: &validateTokenMock{
				request: &authv1.ValidateTokenRequest{
					Token: "token-42",
				},
				response: &authv1.ValidateTokenResponse{
					UserId: "user-42",
				},
				err: assert.AnError,
			},
			wantErr: assert.AnError,
		},
		{
			name: "missing identity",
			clientValidateToken: &validateTokenMock{
				request: &authv1.ValidateTokenRequest{
					Token: "token-42",
				},
			},
			wantErr: auth.ErrInvalidIdentity,
		},
		{
			name: "identity cannot inject metadata",
			clientValidateToken: &validateTokenMock{
				request: &authv1.ValidateTokenRequest{
					Token: "token-42",
				},
				response: &authv1.ValidateTokenResponse{
					UserId: "user\n42",
				},
			},
			wantErr: auth.ErrInvalidIdentity,
		},
		{
			name:          "earlier parent deadline",
			parentTimeout: time.Minute,
			clientValidateToken: &validateTokenMock{
				request: &authv1.ValidateTokenRequest{
					Token: "token-42",
				},
				response: &authv1.ValidateTokenResponse{
					UserId: "user-42",
				},
			},
			want: auth.Identity{
				UserID: "user-42",
			},
		},
		{
			name:             "parent canceled during call",
			cancelDuringCall: true,
			clientValidateToken: &validateTokenMock{
				request: &authv1.ValidateTokenRequest{
					Token: "token-42",
				},
				err: context.Canceled,
			},
			wantContextErr: context.Canceled,
			wantErr:        context.Canceled,
		},
		{
			name:          "expired parent deadline",
			parentTimeout: -time.Second,
			clientValidateToken: &validateTokenMock{
				request: &authv1.ValidateTokenRequest{
					Token: "token-42",
				},
				err: context.DeadlineExceeded,
			},
			wantContextErr: context.DeadlineExceeded,
			wantErr:        context.DeadlineExceeded,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tt.parentTimeout != 0 {
				var cancelDeadline context.CancelFunc
				ctx, cancelDeadline = context.WithTimeout(ctx, tt.parentTimeout)
				defer cancelDeadline()
			}
			var callCtx context.Context
			var calledAt time.Time
			client := mocks.NewAuthenticationClientMock(t)
			if call := tt.clientValidateToken; call != nil {
				client.ValidateTokenMock.Expect(minimock.AnyContext, call.request).Return(call.response, call.err)
				client.ValidateTokenMock.Times(1).Inspect(func(ctx context.Context, _ *authv1.ValidateTokenRequest, _ ...grpc.CallOption) {
					callCtx = ctx
					calledAt = time.Now()
					if tt.cancelDuringCall {
						cancel()
					}
					assert.ErrorIs(t, ctx.Err(), tt.wantContextErr)
				})
			}
			verifier := &Authentication{
				client:  client,
				timeout: time.Hour,
			}
			before := time.Now()
			got, err := verifier.Verify(ctx, "token-42")
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
			if assert.NotNil(t, callCtx) {
				deadline, ok := callCtx.Deadline()
				assert.True(t, ok)
				if tt.parentTimeout != 0 {
					parentDeadline, _ := ctx.Deadline()
					assert.Equal(t, parentDeadline, deadline)
				} else {
					assert.False(t, deadline.Before(before.Add(time.Hour)))
					assert.False(t, deadline.After(calledAt.Add(time.Hour)))
				}
				assert.Error(t, callCtx.Err(), "контекст вызова освобождается после ответа")
			}
		})
	}
}
