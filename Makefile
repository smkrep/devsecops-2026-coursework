API_PROTO_FILES := $(wildcard api/proto/*/v1/*.proto)
THIRD_PARTY_PROTO_FILES := $(wildcard third_party/*/v1/*.proto)
export PATH := $(shell go env GOPATH)/bin:$(PATH)

.PHONY: generate generate-mocks build test run

generate:
	protoc -I api/proto -I third_party \
		--go_out=. --go_opt=module=github.com/smkrep/devsecops-2026-coursework \
		--go-grpc_out=. --go-grpc_opt=module=github.com/smkrep/devsecops-2026-coursework \
		--grpc-gateway_out=. --grpc-gateway_opt=module=github.com/smkrep/devsecops-2026-coursework \
		$(API_PROTO_FILES)
	protoc -I third_party \
		--go_out=. --go_opt=module=github.com/smkrep/devsecops-2026-coursework \
		--go-grpc_out=. --go-grpc_opt=module=github.com/smkrep/devsecops-2026-coursework \
		$(THIRD_PARTY_PROTO_FILES)

generate-mocks:
	go generate ./internal/...

build:
	go build -o bin/gateway ./cmd/gateway

test:
	go test ./...

run:
	go run ./cmd/gateway
