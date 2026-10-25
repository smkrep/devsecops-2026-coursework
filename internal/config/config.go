package config

import (
	"fmt"
	"net"
	"os"
	"time"
)

type Config struct {
	HTTPAddr       string
	GRPCAddr       string
	BackendAddr    string
	BackendTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:    env("HTTP_ADDR", "127.0.0.1:8080"),
		GRPCAddr:    env("GRPC_ADDR", "127.0.0.1:9090"),
		BackendAddr: env("BACKEND_ADDR", "127.0.0.1:9092"),
	}
	for name, addr := range map[string]string{
		"HTTP_ADDR": cfg.HTTPAddr, "GRPC_ADDR": cfg.GRPCAddr, "BACKEND_ADDR": cfg.BackendAddr,
	} {
		host, port, err := net.SplitHostPort(addr)
		if err != nil || host == "" || port == "" {
			return Config{}, fmt.Errorf("%s must be a host:port address with a nonempty host and port", name)
		}
	}
	var err error
	cfg.BackendTimeout, err = time.ParseDuration(env("BACKEND_TIMEOUT", "3s"))
	if err != nil || cfg.BackendTimeout <= 0 {
		return Config{}, fmt.Errorf("BACKEND_TIMEOUT must be a positive duration, for example 3s")
	}
	return cfg, nil
}

func env(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	return fallback
}
