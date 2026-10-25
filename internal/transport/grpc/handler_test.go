package grpctransport

import (
	"context"
	"testing"

	catalogv1 "github.com/smkrep/devsecops-2026-coursework/gen/catalog/v1"
	"github.com/smkrep/devsecops-2026-coursework/internal/gateway"
	"github.com/smkrep/devsecops-2026-coursework/internal/gateway/mocks"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestHandlerGetItem(t *testing.T) {
	type getItemMock struct {
		id   string
		item gateway.Item
		err  error
	}

	tests := []struct {
		name           string
		request        *catalogv1.GetItemRequest
		backendGetItem *getItemMock
		want           *catalogv1.Item
		wantErr        error
	}{
		{
			name: "success",
			request: &catalogv1.GetItemRequest{
				Id: "42",
			},
			backendGetItem: &getItemMock{
				id: "42",
				item: gateway.Item{
					ID:   "42",
					Name: "Book",
				},
			},
			want: &catalogv1.Item{
				Id:   "42",
				Name: "Book",
			},
		},
		{
			name: "unexpected error discards item",
			request: &catalogv1.GetItemRequest{
				Id: "42",
			},
			backendGetItem: &getItemMock{
				id: "42",
				item: gateway.Item{
					ID: "42",
				},
				err: assert.AnError,
			},
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			type contextKey struct{}
			ctx := context.WithValue(t.Context(), contextKey{}, "request-42")
			backend := mocks.NewCatalogBackendMock(t)
			if call := tt.backendGetItem; call != nil {
				backend.GetItemMock.Expect(ctx, call.id).Return(call.item, call.err)
			}

			handler := NewHandler(gateway.NewService(backend))
			got, err := handler.GetItem(ctx, tt.request)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.True(t, proto.Equal(tt.want, got), "GetItem() = %v, want %v", got, tt.want)
		})
	}
}
