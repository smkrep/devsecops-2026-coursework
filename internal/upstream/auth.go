package upstream

import (
	"context"
	"time"

	authv1 "github.com/smkrep/devsecops-2026-coursework/gen/auth/v1"
	"github.com/smkrep/devsecops-2026-coursework/internal/auth"

	"google.golang.org/grpc"
)

// AuthenticationClient описывает RPC проверки токена, используемый исходящим адаптером.
//
//go:generate go tool minimock -i AuthenticationClient -o mocks/authentication_client_mock.go -p mocks -g
type AuthenticationClient interface {
	ValidateToken(ctx context.Context, request *authv1.ValidateTokenRequest, opts ...grpc.CallOption) (*authv1.ValidateTokenResponse, error)
}

type Authentication struct {
	client  AuthenticationClient
	timeout time.Duration
}

func NewAuthentication(conn grpc.ClientConnInterface, timeout time.Duration) *Authentication {
	return &Authentication{
		client:  authv1.NewAuthenticationServiceClient(conn),
		timeout: timeout,
	}
}

func (a *Authentication) Verify(ctx context.Context, token string) (auth.Identity, error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	response, err := a.client.ValidateToken(ctx, &authv1.ValidateTokenRequest{
		Token: token,
	})
	if err != nil {
		return auth.Identity{}, err
	}
	identity := auth.Identity{
		UserID: response.GetUserId(),
	}
	if !auth.ValidIdentity(identity) {
		return auth.Identity{}, auth.ErrInvalidIdentity
	}
	return identity, nil
}
