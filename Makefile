# Makefile for WhatsApp MCP Server

# Variables
BINARY_NAME=whatsapp-mcp-server
BUILD_DIR=build
DOCKER_IMAGE=whatsapp-mcp-server
DOCKER_TAG=latest

# Default target
.PHONY: all
all: build

# Build the application
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=1 go build -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Run the application
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME)..."
	@./$(BUILD_DIR)/$(BINARY_NAME)

# Run with live reload (requires air)
.PHONY: dev
dev:
	@echo "Starting development server with live reload..."
	@air

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f whatsapp.db whatsapp.db_messages.db
	@rm -rf media qr_codes
	@echo "Clean complete"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Lint code
.PHONY: lint
lint:
	@echo "Linting code..."
	@golangci-lint run

# Download dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Update dependencies
.PHONY: update-deps
update-deps:
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy

# Build Docker image
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

# Run Docker container
.PHONY: docker-run
docker-run: docker-build
	@echo "Running Docker container..."
	@docker run -p 8080:8080 \
		-v $(PWD)/data:/app/data \
		-v $(PWD)/media:/app/media \
		-v $(PWD)/qr_codes:/app/qr_codes \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

# Stop Docker container
.PHONY: docker-stop
docker-stop:
	@echo "Stopping Docker container..."
	@docker stop $(DOCKER_IMAGE) || true

# Remove Docker container
.PHONY: docker-rm
docker-rm:
	@echo "Removing Docker container..."
	@docker rm $(DOCKER_IMAGE) || true

# Remove Docker image
.PHONY: docker-rmi
docker-rmi:
	@echo "Removing Docker image..."
	@docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) || true

# Docker compose up
.PHONY: docker-compose-up
docker-compose-up:
	@echo "Starting services with Docker Compose..."
	@docker-compose up -d

# Docker compose down
.PHONY: docker-compose-down
docker-compose-down:
	@echo "Stopping services with Docker Compose..."
	@docker-compose down

# Docker compose logs
.PHONY: docker-compose-logs
docker-compose-logs:
	@echo "Showing Docker Compose logs..."
	@docker-compose logs -f

# Install development tools
.PHONY: install-tools
install-tools:
	@echo "Installing development tools..."
	@go install github.com/cosmtrek/air@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Development tools installed"

# Create directories
.PHONY: setup-dirs
setup-dirs:
	@echo "Creating necessary directories..."
	@mkdir -p data media qr_codes
	@echo "Directories created"

# Setup development environment
.PHONY: setup
setup: setup-dirs deps install-tools
	@echo "Development environment setup complete"

# Generate OpenAPI documentation
.PHONY: openapi
openapi:
	@echo "Generating OpenAPI documentation..."
	@swag init -g main.go -o docs --parseDependency --parseInternal
	@mv docs/swagger.json docs/openapi.json
	@mv docs/swagger.yaml docs/openapi.yaml
	@echo "OpenAPI documentation generated in docs/"

# Generate documentation
.PHONY: docs
docs:
	@echo "Generating documentation..."
	@godoc -http=:6060 &
	@echo "Documentation server started at http://localhost:6060"

# Security scan
.PHONY: security
security:
	@echo "Running security scan..."
	@gosec ./...

# Benchmark tests
.PHONY: benchmark
benchmark:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          - Build the application"
	@echo "  run            - Run the application"
	@echo "  dev            - Run with live reload"
	@echo "  clean          - Clean build artifacts"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code"
	@echo "  deps           - Download dependencies"
	@echo "  update-deps    - Update dependencies"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run Docker container"
	@echo "  docker-stop    - Stop Docker container"
	@echo "  docker-rm      - Remove Docker container"
	@echo "  docker-rmi     - Remove Docker image"
	@echo "  docker-compose-up    - Start with Docker Compose"
	@echo "  docker-compose-down  - Stop Docker Compose"
	@echo "  docker-compose-logs  - Show Docker Compose logs"
	@echo "  install-tools  - Install development tools"
	@echo "  setup-dirs     - Create necessary directories"
	@echo "  setup          - Setup development environment"
	@echo "  openapi        - Generate OpenAPI documentation"
	@echo "  docs           - Generate documentation"
	@echo "  security       - Run security scan"
	@echo "  benchmark      - Run benchmarks"
	@echo "  help           - Show this help"
