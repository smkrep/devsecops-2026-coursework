package gateway_test

import (
	"testing"

	"coursework/gateway/internal/gateway"
	"coursework/gateway/internal/gateway/mocks"

	"github.com/stretchr/testify/assert"
)

func TestServiceGetItem(t *testing.T) {
	type getItemMock struct {
		id   string
		item gateway.Item
		err  error
	}

	tests := []struct {
		name           string
		id             string
		backendGetItem *getItemMock
		want           gateway.Item
		wantErr        error
	}{
		{
			name: "success",
			id:   "42",
			backendGetItem: &getItemMock{
				id: "42",
				item: gateway.Item{
					ID:   "42",
					Name: "Book",
				},
			},
			want: gateway.Item{
				ID:   "42",
				Name: "Book",
			},
		},
		{
			name: "backend error",
			id:   "42",
			backendGetItem: &getItemMock{
				id:  "42",
				err: assert.AnError,
			},
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			backend := mocks.NewCatalogBackendMock(t)
			if call := tt.backendGetItem; call != nil {
				backend.GetItemMock.Expect(ctx, call.id).Return(call.item, call.err)
			}

			got, err := gateway.NewService(backend).GetItem(ctx, tt.id)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
		})
	}
}
