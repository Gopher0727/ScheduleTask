.PHONY: build run fmt fmt-check vet test integration check proto

build:
	go build -trimpath -o bin/scheduletask ./cmd/scheduletask

run:
	go run ./cmd/scheduletask -config config.toml

fmt:
	gofmt -w cmd internal

fmt-check:
	@test -z "$$(gofmt -l cmd internal)" || (gofmt -l cmd internal; exit 1)

vet:
	go vet ./...

test:
	go test -race -count=1 ./...

integration:
	@test -n "$(SCHEDULETASK_TEST_POSTGRES_DSN)" || (echo "Set SCHEDULETASK_TEST_POSTGRES_DSN to a disposable database"; exit 1)
	@test -n "$(SCHEDULETASK_TEST_REDIS_ADDR)" || (echo "Set SCHEDULETASK_TEST_REDIS_ADDR to a disposable Redis instance"; exit 1)
	go test -race -count=1 ./...

check: fmt-check vet test

# Install protoc and the pinned Go plugins described in CONTRIBUTING.md first.
proto:
	protoc -I api/proto --go_out=api --go-grpc_out=api api/proto/xtimer.proto
