.PHONY: build build-api build-worker clean run-api run-worker tidy

# Build output directory
BIN_DIR := bin

# Build both binaries
build: build-api build-worker

build-api:
	go build -o $(BIN_DIR)/hls-api ./cmd/api

build-worker:
	go build -o $(BIN_DIR)/hls-worker ./cmd/worker

# Run locally
run-api:
	go run ./cmd/api -config configs/config.yaml

run-worker:
	go run ./cmd/worker -config configs/config.yaml -api http://localhost:8080

# Clean build artifacts
clean:
	rm -rf $(BIN_DIR)

# Tidy dependencies
tidy:
	go mod tidy

# Build for Linux (production deployment)
build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(BIN_DIR)/hls-api-linux ./cmd/api
	GOOS=linux GOARCH=amd64 go build -o $(BIN_DIR)/hls-worker-linux ./cmd/worker
