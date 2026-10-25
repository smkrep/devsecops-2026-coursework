package upstream

import (
	"context"
	"testing"
	"time"

	catalogv1 "github.com/smkrep/devsecops-2026-coursework/gen/catalog/v1"
	"github.com/smkrep/devsecops-2026-coursework/internal/gateway"
	"github.com/smkrep/devsecops-2026-coursework/internal/upstream/mocks"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestCatalogGetItem(t *testing.T) {
	type getItemMock struct {
		request *catalogv1.GetItemRequest
		item    *catalogv1.Item
		err     error
	}

	tests := []struct {
		name          string
		id            string
		clientGetItem *getItemMock
		want          gateway.Item
		wantErr       error
	}{
		{
			name: "success",
			id:   "42",
			clientGetItem: &getItemMock{
				request: &catalogv1.GetItemRequest{
					Id: "42",
				},
				item: &catalogv1.Item{
					Id:   "42",
					Name: "Book",
				},
			},
			want: gateway.Item{
				ID:   "42",
				Name: "Book",
			},
		},
		{
			name: "error discards response",
			id:   "42",
			clientGetItem: &getItemMock{
				request: &catalogv1.GetItemRequest{
					Id: "42",
				},
				item: &catalogv1.Item{
					Id:   "42",
					Name: "Book",
				},
				err: assert.AnError,
			},
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := metadata.NewOutgoingContext(t.Context(), metadata.Pairs("x-request-id", "request-42", "x-user-id", "user-42"))
			var callCtx context.Context
			client := mocks.NewCatalogClientMock(t)
			if call := tt.clientGetItem; call != nil {
				client.GetItemMock.Expect(minimock.AnyContext, call.request).Return(call.item, call.err)
				client.GetItemMock.Times(1).Inspect(func(ctx context.Context, _ *catalogv1.GetItemRequest, _ ...grpc.CallOption) {
					callCtx = ctx
					md, _ := metadata.FromOutgoingContext(ctx)
					assert.Equal(t, []string{
						"user-42",
					}, md.Get("x-user-id"))
					assert.Equal(t, []string{
						"request-42",
					}, md.Get("x-request-id"))
				})
			}

			catalog := &Catalog{
				client:  client,
				timeout: time.Hour,
			}
			got, err := catalog.GetItem(ctx, tt.id)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
			if assert.NotNil(t, callCtx, "backend was not called") {
				assert.ErrorIs(t, callCtx.Err(), context.Canceled)
			}
		})
	}
}

func TestCatalogGetItemDeadline(t *testing.T) {
	type getItemMock struct {
		request *catalogv1.GetItemRequest
		item    *catalogv1.Item
		err     error
	}

	tests := []struct {
		name           string
		parentTimeout  time.Duration
		backendTimeout time.Duration
		clientGetItem  *getItemMock
		wantParent     bool
	}{
		{
			name:           "backend timeout limits call",
			parentTimeout:  2 * time.Hour,
			backendTimeout: time.Hour,
			clientGetItem: &getItemMock{
				request: &catalogv1.GetItemRequest{
					Id: "42",
				},
				item: &catalogv1.Item{},
			},
		},
		{
			name:           "earlier parent deadline is preserved",
			parentTimeout:  time.Hour,
			backendTimeout: 2 * time.Hour,
			clientGetItem: &getItemMock{
				request: &catalogv1.GetItemRequest{
					Id: "42",
				},
				item: &catalogv1.Item{},
			},
			wantParent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(t.Context(), tt.parentTimeout)
			defer cancel()
			parentDeadline, _ := ctx.Deadline()
			var callCtx context.Context
			var calledAt time.Time
			client := mocks.NewCatalogClientMock(t)
			if call := tt.clientGetItem; call != nil {
				client.GetItemMock.Expect(minimock.AnyContext, call.request).Return(call.item, call.err)
				client.GetItemMock.Times(1).Inspect(func(ctx context.Context, _ *catalogv1.GetItemRequest, _ ...grpc.CallOption) {
					callCtx = ctx
					calledAt = time.Now()
				})
			}

			catalog := &Catalog{
				client:  client,
				timeout: tt.backendTimeout,
			}
			before := time.Now()
			_, err := catalog.GetItem(ctx, "42")
			assert.NoError(t, err)
			assert.NoError(t, ctx.Err(), "parent context must remain active")
			if !assert.NotNil(t, callCtx) {
				return
			}
			deadline, ok := callCtx.Deadline()
			if !assert.True(t, ok, "backend context must have a deadline") {
				return
			}
			if tt.wantParent {
				assert.Equal(t, parentDeadline, deadline)
			} else {
				assert.False(t, deadline.Before(before.Add(tt.backendTimeout)), "backend deadline is too early")
				assert.False(t, deadline.After(calledAt.Add(tt.backendTimeout)), "backend deadline is too late")
			}
		})
	}
}

func TestCatalogGetItemCancellation(t *testing.T) {
	type getItemMock struct {
		request *catalogv1.GetItemRequest
		item    *catalogv1.Item
		err     error
	}

	tests := []struct {
		name           string
		backendTimeout time.Duration
		clientGetItem  *getItemMock
		wantErr        error
	}{
		{
			name:           "parent canceled during call",
			backendTimeout: time.Hour,
			clientGetItem: &getItemMock{
				request: &catalogv1.GetItemRequest{
					Id: "42",
				},
				err: context.Canceled,
			},
			wantErr: context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			client := mocks.NewCatalogClientMock(t)
			if call := tt.clientGetItem; call != nil {
				client.GetItemMock.Expect(minimock.AnyContext, call.request).Return(call.item, call.err)
				client.GetItemMock.Times(1).Inspect(func(ctx context.Context, _ *catalogv1.GetItemRequest, _ ...grpc.CallOption) {
					cancel()
					assert.ErrorIs(t, ctx.Err(), tt.wantErr)
				})
			}

			catalog := &Catalog{
				client:  client,
				timeout: tt.backendTimeout,
			}
			got, err := catalog.GetItem(ctx, "42")
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Empty(t, got)
		})
	}
}
