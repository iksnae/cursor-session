.PHONY: build install clean test

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

# Show help
help:
	@echo "Available targets:"
	@echo "  make build      - Build the binary"
	@echo "  make install    - Install to system (/usr/local/bin or ~/.local/bin)"
	@echo "  make go-install - Install using 'go install' (recommended)"
	@echo "  make clean      - Remove build artifacts"
	@echo "  make test       - Run tests"






