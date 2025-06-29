.PHONY: help build run test clean docker-build docker-up docker-down logs deps

help:
	@echo "Available commands:"
	@echo "  build        - Build the application"
	@echo "  run          - Run the application locally"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  deps         - Download dependencies"
	@echo "  docker-build - Build Docker image"
	@echo "  fmt          - Format Go code"
	@echo "  lint         - Run linter"

build:
	@echo "Building application..."
	@go build -o bin/memebot .

run:
	@echo "Running application..."
	@go run main.go

test:
	@echo "Running tests..."
	@go test -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -cover ./...

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@go clean

deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

fmt:
	@echo "Formatting Go code..."
	@go fmt ./...

lint:
	@echo "Running linter..."
	@golangci-lint run

docker-build:
	@echo "Building Docker image..."
	@docker build -t memebot .

install-tools:
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

release:
	@echo "Building release version..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o bin/memebot-linux-amd64 .
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-w -s" -o bin/memebot-darwin-amd64 .
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-w -s" -o bin/memebot-windows-amd64.exe .