.PHONY: build install clean test test-coverage test-coverage-html test-coverage-check docker-dev docker-test docker-build docker-shell docker-clean docker-reset

# Build the binary
build:
	@echo "Building cursor-session..."
	@go build -buildvcs=false -ldflags "-X 'github.com/iksnae/cursor-session/cmd.version=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)' -X 'github.com/iksnae/cursor-session/cmd.commit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)' -X 'github.com/iksnae/cursor-session/cmd.date=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)'" -o cursor-session .

# Install to system (requires sudo for /usr/local/bin)
install: build
	@echo "Installing cursor-session..."
	@if [ "$(shell id -u)" -eq 0 ]; then \
		cp cursor-session /usr/local/bin/; \
		echo "Installed to /usr/local/bin/cursor-session"; \
	else \
		mkdir -p ~/.local/bin; \
		cp cursor-session ~/.local/bin/; \
		echo "Installed to ~/.local/bin/cursor-session"; \
		echo "Make sure ~/.local/bin is in your PATH"; \
	fi

# Install using go install (recommended)
go-install:
	@echo "Installing using 'go install'..."
	@go install -buildvcs=false -ldflags "-X 'github.com/iksnae/cursor-session/cmd.version=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)' -X 'github.com/iksnae/cursor-session/cmd.commit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)' -X 'github.com/iksnae/cursor-session/cmd.date=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)'" .

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f cursor-session
	@go clean

# Run tests
test:
	@go test ./... -v

# Generate coverage profile and display summary
test-coverage:
	@echo "Running tests with coverage..."
	@go test ./... -coverprofile=coverage.out -covermode=atomic
	@go tool cover -func=coverage.out | tail -1

# Generate HTML coverage report
test-coverage-html: test-coverage
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Check coverage against 80% threshold
test-coverage-check: test-coverage
	@coverage=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Total coverage: $$coverage%"; \
	if [ $$(echo "$$coverage < 80" | bc -l 2>/dev/null || echo "1") -eq 1 ]; then \
		echo "❌ Coverage $$coverage% is below 80% threshold"; \
		exit 1; \
	else \
		echo "✅ Coverage $$coverage% meets 80% threshold"; \
	fi

# Docker Development Targets

# Start development container with cursor-agent
docker-dev:
	@echo "Starting development container..."
	@docker-compose up -d dev
	@echo "Development container started. Use 'make docker-shell' to access it."

# Run tests in Docker container
docker-test:
	@echo "Running tests in Docker container..."
	@docker-compose run --rm test

# Build binary in Docker container
docker-build:
	@echo "Building binary in Docker container..."
	@docker-compose run --rm build
	@echo "Build complete. Binary available at ./cursor-session"

# Access interactive shell in development container
docker-shell:
	@echo "Opening shell in development container..."
	@docker-compose exec dev /bin/bash || docker-compose run --rm dev /bin/bash

# Clean Docker artifacts (containers, images, volumes)
docker-clean:
	@echo "Cleaning Docker artifacts..."
	@docker-compose down -v
	@docker-compose rm -f
	@echo "Docker artifacts cleaned"

# Clean Docker environment (containers, networks, volumes)
# Note: Storage is ephemeral (like GitHub Actions) - no persistent volumes to reset
docker-reset:
	@echo "Cleaning Docker environment..."
	@docker-compose down -v 2>/dev/null || true
	@echo "Docker environment cleaned. Run 'make docker-dev' to start fresh."

# Show help
help:
	@echo "Available targets:"
	@echo "  make build              - Build the binary"
	@echo "  make install            - Install to system (/usr/local/bin or ~/.local/bin)"
	@echo "  make go-install         - Install using 'go install' (recommended)"
	@echo "  make clean              - Remove build artifacts"
	@echo "  make test               - Run tests"
	@echo "  make test-coverage      - Generate coverage profile and display summary"
	@echo "  make test-coverage-html - Generate HTML coverage report"
	@echo "  make test-coverage-check - Check coverage against 80% threshold"
	@echo ""
	@echo "Docker targets:"
	@echo "  make docker-dev         - Start development container with cursor-agent"
	@echo "  make docker-test        - Run tests in Docker container"
	@echo "  make docker-build       - Build binary in Docker container"
	@echo "  make docker-shell       - Access interactive shell in development container"
	@echo "  make docker-clean       - Clean Docker artifacts (containers, images, volumes)"
	@echo "  make docker-reset       - Clean Docker environment (containers, networks, volumes)"
