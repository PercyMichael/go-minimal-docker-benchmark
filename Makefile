.PHONY: build run dev test generate

# Default target
all: build

# Build production binary
build:
	export PATH=$$PATH:/usr/local/go/bin; go build -o bin/api ./cmd/api

# Run local API server
run:
	export PATH=$$PATH:/usr/local/go/bin; go run ./cmd/api

# Run tests
test:
	export PATH=$$PATH:/usr/local/go/bin; go test -v ./...

# Generate sqlc code
generate:
	sqlc generate
