package upstream

import (
	"context"
	"testing"
	"time"

	ratelimitv1 "github.com/smkrep/devsecops-2026-coursework/gen/ratelimit/v1"
	"github.com/smkrep/devsecops-2026-coursework/internal/upstream/mocks"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestRateLimitAllow(t *testing.T) {
	type checkLimitMock struct {
		request  *ratelimitv1.CheckLimitRequest
		response *ratelimitv1.CheckLimitResponse
		err      error
	}

	request := &ratelimitv1.CheckLimitRequest{
		UserId:   "user-42",
		Resource: "/catalog.v1.CatalogService/GetItem",
	}
	tests := []struct {
		name             string
		clientCheckLimit *checkLimitMock
		parentTimeout    time.Duration
		cancelDuringCall bool
		wantContextErr   error
		want             bool
		wantErr          error
	}{
		{
			name: "allowed request",
			clientCheckLimit: &checkLimitMock{
				request: request,
				response: &ratelimitv1.CheckLimitResponse{
					Allowed: true,
				},
			},
			want: true,
		},
		{
			name: "denied request",
			clientCheckLimit: &checkLimitMock{
				request:  request,
				response: &ratelimitv1.CheckLimitResponse{},
			},
		},
		{
			name: "error discards allow decision",
			clientCheckLimit: &checkLimitMock{
				request: request,
				response: &ratelimitv1.CheckLimitResponse{
					Allowed: true,
				},
				err: assert.AnError,
			},
			wantErr: assert.AnError,
		},
		{
			name: "missing response",
			clientCheckLimit: &checkLimitMock{
				request: request,
			},
			wantErr: ErrInvalidRateLimitResponse,
		},
		{
			name:          "earlier parent deadline",
			parentTimeout: time.Minute,
			clientCheckLimit: &checkLimitMock{
				request: request,
				response: &ratelimitv1.CheckLimitResponse{
					Allowed: true,
				},
			},
			want: true,
		},
		{
			name:             "parent canceled during call",
			cancelDuringCall: true,
			clientCheckLimit: &checkLimitMock{
				request: request,
				err:     context.Canceled,
			},
			wantContextErr: context.Canceled,
			wantErr:        context.Canceled,
		},
		{
			name:          "expired parent deadline",
			parentTimeout: -time.Second,
			clientCheckLimit: &checkLimitMock{
				request: request,
				err:     context.DeadlineExceeded,
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
			ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("x-user-id", "user-42"))
			var callCtx context.Context
			var calledAt time.Time
			client := mocks.NewRateLimitClientMock(t)
			if call := tt.clientCheckLimit; call != nil {
				client.CheckLimitMock.Expect(minimock.AnyContext, call.request).Return(call.response, call.err)
				client.CheckLimitMock.Times(1).Inspect(func(ctx context.Context, _ *ratelimitv1.CheckLimitRequest, _ ...grpc.CallOption) {
					callCtx = ctx
					calledAt = time.Now()
					if tt.cancelDuringCall {
						cancel()
					}
					assert.ErrorIs(t, ctx.Err(), tt.wantContextErr)
					md, _ := metadata.FromOutgoingContext(ctx)
					assert.Equal(t, []string{
						"user-42",
					}, md.Get("x-user-id"))
				})
			}
			limiter := &RateLimit{
				client:  client,
				timeout: time.Hour,
			}
			before := time.Now()
			got, err := limiter.Allow(ctx, "user-42", "/catalog.v1.CatalogService/GetItem")
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
				assert.Error(t, callCtx.Err(), "контекст проверки освобождается после ответа")
			}
		})
	}
}
