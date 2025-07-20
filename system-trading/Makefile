# System Trading Makefile

.PHONY: help build test run clean docker-build docker-up docker-down logs

# Default target
help:
	@echo "Available targets:"
	@echo "  build        - Build the Go application"
	@echo "  test         - Run tests"
	@echo "  run          - Run the application locally"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-up    - Start all services with Docker Compose"
	@echo "  docker-down  - Stop all Docker services"
	@echo "  logs         - Show Docker logs"

# Build the application
build:
	@echo "Building system-trading..."
	@mkdir -p bin
	@go build -o bin/system-trading ./cmd/main.go
	@echo "Build complete: bin/system-trading"

# Run tests
test:
	@echo "Running tests..."
	@go test -v -race -cover ./...

# Run the application locally
run: build
	@echo "Starting system-trading..."
	@./bin/system-trading

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@go clean

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	@docker build -t system-trading:latest .

# Start all services with Docker Compose
docker-up:
	@echo "Starting all services..."
	@docker-compose up -d
	@echo "Services started. Check status with: docker-compose ps"
	@echo ""
	@echo "Available services:"
	@echo "  Trading Core:  http://localhost:8080"
	@echo "  Prometheus:    http://localhost:9090"
	@echo "  Grafana:       http://localhost:3000 (admin/admin)"
	@echo "  NATS Monitor:  http://localhost:8222"

# Stop all Docker services
docker-down:
	@echo "Stopping all services..."
	@docker-compose down

# Show Docker logs
logs:
	@docker-compose logs -f trading-core

# Format Go code
fmt:
	@echo "Formatting Go code..."
	@go fmt ./...

# Run Go linter
lint:
	@echo "Running Go linter..."
	@golangci-lint run

# Tidy Go modules
tidy:
	@echo "Tidying Go modules..."
	@go mod tidy

# Generate Go code
generate:
	@echo "Generating Go code..."
	@go generate ./...

# Run all checks (format, lint, test)
check: fmt lint test
	@echo "All checks passed!"

# Development setup
dev-setup:
	@echo "Setting up development environment..."
	@go mod download
	@echo "Development setup complete!"

# View application logs in development
dev-logs:
	@echo "Showing application logs..."
	@tail -f /tmp/system-trading.log 2>/dev/null || echo "No log file found. Run the application first."