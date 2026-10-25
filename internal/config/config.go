package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidAddress = errors.New("invalid address")
	ErrInvalidTimeout = errors.New("invalid backend timeout")
)

type Config struct {
	HTTPAddr              string
	GRPCAddr              string
	CatalogBackendAddrs   []string
	OrdersBackendAddrs    []string
	BackendTimeout        time.Duration
	AuthBackendAddrs      []string
	AuthTimeout           time.Duration
	RateLimitBackendAddrs []string
	RateLimitTimeout      time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr: env("HTTP_ADDR", "127.0.0.1:8080"),
		GRPCAddr: env("GRPC_ADDR", "127.0.0.1:9090"),
	}
	for name, addr := range map[string]string{
		"HTTP_ADDR": cfg.HTTPAddr, "GRPC_ADDR": cfg.GRPCAddr,
	} {
		host, port, err := net.SplitHostPort(addr)
		if err != nil || host == "" || port == "" {
			return Config{}, fmt.Errorf("%w: %s must be a host:port address with a nonempty host and port", ErrInvalidAddress, name)
		}
	}
	var err error
	cfg.CatalogBackendAddrs, err = backendAddresses("CATALOG_BACKEND_ADDRS", "127.0.0.1:9092")
	if err != nil {
		return Config{}, err
	}
	cfg.OrdersBackendAddrs, err = backendAddresses("ORDERS_BACKEND_ADDRS", "127.0.0.1:9093")
	if err != nil {
		return Config{}, err
	}
	cfg.AuthBackendAddrs, err = backendAddresses("AUTH_BACKEND_ADDRS", "127.0.0.1:9096")
	if err != nil {
		return Config{}, err
	}
	cfg.RateLimitBackendAddrs, err = backendAddresses("RATE_LIMIT_BACKEND_ADDRS", "127.0.0.1:9098")
	if err != nil {
		return Config{}, err
	}
	cfg.RateLimitTimeout, err = time.ParseDuration(env("RATE_LIMIT_TIMEOUT", "1s"))
	if err != nil || cfg.RateLimitTimeout <= 0 {
		return Config{}, fmt.Errorf("%w: RATE_LIMIT_TIMEOUT must be a positive duration, for example 1s", ErrInvalidTimeout)
	}
	cfg.AuthTimeout, err = time.ParseDuration(env("AUTH_TIMEOUT", "1s"))
	if err != nil || cfg.AuthTimeout <= 0 {
		return Config{}, fmt.Errorf("%w: AUTH_TIMEOUT must be a positive duration, for example 1s", ErrInvalidTimeout)
	}
	cfg.BackendTimeout, err = time.ParseDuration(env("BACKEND_TIMEOUT", "3s"))
	if err != nil || cfg.BackendTimeout <= 0 {
		return Config{}, fmt.Errorf("%w: BACKEND_TIMEOUT must be a positive duration, for example 3s", ErrInvalidTimeout)
	}
	return cfg, nil
}

func env(name string, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	return fallback
}

// backendAddresses читает разделённый запятыми список адресов host:port и удаляет повторы.
func backendAddresses(name string, fallback string) ([]string, error) {
	var addresses []string
	seen := make(map[string]bool)
	for _, value := range strings.Split(env(name, fallback), ",") {
		addr := strings.TrimSpace(value)
		host, port, err := net.SplitHostPort(addr)
		number, portErr := strconv.Atoi(port)
		if err != nil || host == "" || strings.ContainsAny(host, " /\t\n\r") || portErr != nil || number < 1 || number > 65535 {
			return nil, fmt.Errorf("%w: %s must contain comma-separated host:port addresses with ports between 1 and 65535: %q", ErrInvalidAddress, name, value)
		}
		if !seen[addr] {
			addresses = append(addresses, addr)
			seen[addr] = true
		}
	}
	return addresses, nil
}
