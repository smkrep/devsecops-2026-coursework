PROTO_FILES := api/proto/catalog/v1/catalog.proto
export PATH := $(shell go env GOPATH)/bin:$(PATH)

.PHONY: generate build test run

generate:
	protoc -I api/proto -I third_party \
		--go_out=. --go_opt=module=coursework/gateway \
		--go-grpc_out=. --go-grpc_opt=module=coursework/gateway \
		--grpc-gateway_out=. --grpc-gateway_opt=module=coursework/gateway \
		$(PROTO_FILES)

build:
	go build -o bin/gateway ./cmd/gateway

test:
	go test ./...

run:
	go run ./cmd/gateway
