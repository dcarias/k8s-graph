# KubeGraph Makefile
# Provides convenient targets for building, testing, and Docker operations

.PHONY: help build test clean docker-build docker-push docker-run docker-build-in-docker cli cli-build cli-test

# Default target
help:
	@echo "KubeGraph - Available targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  build          Build the KubeGraph binary"
	@echo "  cli            Build the kubegraph-cli binary"
	@echo "  test           Run tests"
	@echo "  clean          Clean build artifacts"
	@echo ""
	@echo "Docker targets (local build):"
	@echo "  docker-build   Build Docker image with locally built binary (default)"
	@echo "  docker-push    Build and push Docker image with locally built binary"
	@echo "  docker-run     Run Docker container locally"
	@echo ""
	@echo "Docker targets (Docker build):"
	@echo "  docker-build-in-docker  Build binary inside Docker (requires local libs)"
	@echo ""
	@echo "Manual Docker build (with options):"
	@echo "  make docker-manual TAG=v1.0.0 PUSH=true"
	@echo "  make docker-manual TAG=v1.0.0 PUSH=true DOCKER_BUILD=true"
	@echo ""

# Build the binary
build:
	@echo "Building KubeGraph..."
	@echo "Git commit: $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")"
	@echo "Git branch: $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")"
	go build -ldflags "-X 'kubegraph/pkg/version.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")' -X 'kubegraph/pkg/version.GitBranch=$(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")'" -o kubegraph .

# Build the CLI binary
cli:
	@echo "Building kubegraph-cli..."
	@echo "Git commit: $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")"
	@echo "Git branch: $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")"
	go build -ldflags "-X 'kubegraph/pkg/version.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")' -X 'kubegraph/pkg/version.GitBranch=$(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")'" -o kubegraph-cli ./cmd/cli

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f kubegraph kubegraph-cli
	go clean

# Docker targets (local build - default)
docker-build:
	@echo "Building Docker image with locally built binary..."
	./scripts/build-docker.sh

docker-push:
	@echo "Building and pushing Docker image with locally built binary..."
	./scripts/build-docker.sh -p

# Docker targets (Docker build)
docker-build-in-docker:
	@echo "Building Docker image with binary built inside Docker..."
	./scripts/build-docker.sh --docker-build

docker-run:
	@echo "Running Docker container..."
	docker run -it --rm \
		-e NEO4J_URI=bolt://localhost:7687 \
		-e NEO4J_USERNAME=neo4j \
		-e NEO4J_PASSWORD=password \
		-e CLUSTER_NAME=my-cluster \
		dcarias/kubegraph:latest

# Manual Docker build with options
docker-manual:
	@echo "Building Docker image with custom options..."
	@if [ -z "$(TAG)" ]; then \
		echo "Using default tag: latest"; \
		if [ "$(DOCKER_BUILD)" = "true" ]; then \
			./scripts/build-docker.sh --docker-build $(if $(PUSH),-p,); \
		else \
			./scripts/build-docker.sh $(if $(PUSH),-p,); \
		fi; \
	else \
		echo "Using tag: $(TAG)"; \
		if [ "$(DOCKER_BUILD)" = "true" ]; then \
			./scripts/build-docker.sh -t $(TAG) --docker-build $(if $(PUSH),-p,); \
		else \
			./scripts/build-docker.sh -t $(TAG) $(if $(PUSH),-p,); \
		fi; \
	fi

# Development targets
dev-build:
	@echo "Building for development..."
	go build -race -o kubegraph .

dev-test:
	@echo "Running tests with race detection..."
	go test -race -v ./...

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run

# Generate documentation
docs:
	@echo "Generating documentation..."
	@if command -v godoc > /dev/null; then \
		echo "Starting godoc server on http://localhost:6060"; \
		godoc -http=:6060; \
	else \
		echo "godoc not found. Install with: go install golang.org/x/tools/cmd/godoc@latest"; \
	fi 
