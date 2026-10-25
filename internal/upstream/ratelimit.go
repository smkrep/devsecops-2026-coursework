package upstream

import (
	"context"
	"errors"
	"time"

	ratelimitv1 "github.com/smkrep/devsecops-2026-coursework/gen/ratelimit/v1"

	"google.golang.org/grpc"
)

var ErrInvalidRateLimitResponse = errors.New("invalid rate limit response")

// RateLimitClient описывает RPC проверки лимита, используемый исходящим адаптером.
//
//go:generate go tool minimock -i RateLimitClient -o mocks/rate_limit_client_mock.go -p mocks -g
type RateLimitClient interface {
	CheckLimit(ctx context.Context, request *ratelimitv1.CheckLimitRequest, opts ...grpc.CallOption) (*ratelimitv1.CheckLimitResponse, error)
}

type RateLimit struct {
	client  RateLimitClient
	timeout time.Duration
}

func NewRateLimit(conn grpc.ClientConnInterface, timeout time.Duration) *RateLimit {
	return &RateLimit{
		client:  ratelimitv1.NewRateLimitServiceClient(conn),
		timeout: timeout,
	}
}

func (r *RateLimit) Allow(ctx context.Context, userID string, resource string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	response, err := r.client.CheckLimit(ctx, &ratelimitv1.CheckLimitRequest{
		UserId:   userID,
		Resource: resource,
	})
	if err != nil {
		return false, err
	}
	if response == nil {
		return false, ErrInvalidRateLimitResponse
	}
	return response.GetAllowed(), nil
}
