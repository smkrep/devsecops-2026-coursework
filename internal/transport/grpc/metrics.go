package grpctransport

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RequestMetrics описывает учёт запроса, необходимый перехватчику метрик.
//
//go:generate go tool minimock -i RequestMetrics -o mocks/request_metrics_mock.go -p mocks -g
type RequestMetrics interface {
	StartRequest(method string) func(string)
}

// Metrics учитывает весь вызов, включая аутентификацию и проверку лимитов.
func Metrics(recorder RequestMetrics) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, request any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		finish := recorder.StartRequest(info.FullMethod)
		code := codes.Internal
		defer func() {
			finish(code.String())
		}()
		response, err := handler(ctx, request)
		code = status.Code(err)
		return response, err
	}
}
