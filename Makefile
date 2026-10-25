PROTO_FILES := api/proto/catalog/v1/catalog.proto
export PATH := $(shell go env GOPATH)/bin:$(PATH)

.PHONY: generate generate-mocks build test run

generate:
	protoc -I api/proto -I third_party \
		--go_out=. --go_opt=module=coursework/gateway \
		--go-grpc_out=. --go-grpc_opt=module=coursework/gateway \
		--grpc-gateway_out=. --grpc-gateway_opt=module=coursework/gateway \
		$(PROTO_FILES)

generate-mocks:
	go tool minimock -i coursework/gateway/internal/gateway.CatalogBackend -o internal/gateway/mocks/catalog_backend_mock.go -p mocks -g
	go tool minimock -i coursework/gateway/gen/catalog/v1.CatalogServiceClient -o internal/upstream/mocks/catalog_service_client_mock.go -p mocks -g

build:
	go build -o bin/gateway ./cmd/gateway

test:
	go test ./...

run:
	go run ./cmd/gateway
