package upstream

import (
	"errors"
	"fmt"
	"slices"

	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
)

var ErrNoBackendAddresses = errors.New("at least one backend address is required")

const backendServiceConfig = `{"loadBalancingConfig":[{"` + roundrobin.Name + `":{}}]}`

// NewBackendConn создаёт клиент с ленивым подключением и балансировкой RPC между готовыми backend.
// Адреса должны быть проверены и иметь вид host:port. У каждого соединения свой resolver.
func NewBackendConn(addresses []string) (*grpc.ClientConn, error) {
	if len(addresses) == 0 {
		return nil, ErrNoBackendAddresses
	}
	r := &backendResolver{
		addresses: slices.Clone(addresses),
	}
	return grpc.NewClient(r.Scheme()+":///backend",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithResolvers(r),
	)
}

// backendResolver передаёт фиксированный набор экземпляров: отдельный endpoint для каждого адреса.
// Повторным подключением и выбором готовых экземпляров управляет gRPC round_robin.
type backendResolver struct {
	addresses []string
}

func (*backendResolver) Scheme() string { return "backend" }

func (r *backendResolver) Build(_ resolver.Target, cc resolver.ClientConn, _ resolver.BuildOptions) (resolver.Resolver, error) {
	config := cc.ParseServiceConfig(backendServiceConfig)
	if config.Err != nil {
		return nil, fmt.Errorf("parse backend service config: %w", config.Err)
	}
	state := resolver.State{
		ServiceConfig: config,
	}
	for _, addr := range r.addresses {
		state.Endpoints = append(state.Endpoints, resolver.Endpoint{
			Addresses: []resolver.Address{
				{
					Addr: addr,
				},
			},
		})
	}
	if err := cc.UpdateState(state); err != nil {
		return nil, fmt.Errorf("update backend endpoints: %w", err)
	}
	return r, nil
}

// Настроенный список адресов сохраняется до перезапуска шлюза.
func (*backendResolver) ResolveNow(resolver.ResolveNowOptions) {}
func (*backendResolver) Close()                                {}
