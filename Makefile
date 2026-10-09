# MCRFlow Broadcast Automation - Development & Build Makefile
#
SHELL := /bin/bash
BIN_DIR := ./bin
TMP_DIR := ./tmp
AIR := $(shell which air 2>/dev/null || echo $(HOME)/go/bin/air)

.PHONY: help install-tools dev dev-control dev-backend dev-agent dev-ui build build-control build-agent test test-unit test-e2e test-race fmt vet tidy clean

# Default target: display help
help:
	@echo "================================================================================"
	@echo "  MCRFlow Developer Commands"
	@echo "================================================================================"
	@echo "  make install-tools   Install live-reload tool 'air' (Go)"
	@echo "  make dev-control     Run Control Plane with live auto-reload (Air)"
	@echo "  make dev-backend     Alias for dev-control"
	@echo "  make dev-agent       Run Edge Playout Agent with live auto-reload (Air)"
	@echo "  make dev-ui          Run Web UI development server on port 3000"
	@echo ""
	@echo "  make build           Compile all binaries into $(BIN_DIR)/"
	@echo "  make build-control   Compile $(BIN_DIR)/mcrflow-control"
	@echo "  make build-agent     Compile $(BIN_DIR)/mcrflow-agent"
	@echo ""
	@echo "  make test            Run all tests across the repository"
	@echo "  make test-unit       Run internal unit tests"
	@echo "  make test-e2e        Run end-to-end integration tests"
	@echo "  make test-race       Run tests with Go race detector"
	@echo ""
	@echo "  make fmt             Format code with 'go fmt'"
	@echo "  make vet             Run static analysis with 'go vet'"
	@echo "  make tidy            Tidy and verify Go module dependencies"
	@echo "  make clean           Remove build artifacts and temporary files"
	@echo "================================================================================"

# Install developer tools (Air for hot reload)
install-tools:
	@echo "--> Installing air live-reload tool..."
	go install github.com/air-verse/air@latest

# Run Control Plane with Air live reload
dev-control:
	@echo "--> Starting MCRFlow Control Plane with Air live reload..."
	@if [ -x "$$(which air 2>/dev/null)" ]; then \
		air -c .air.control.toml; \
	elif [ -x "$$(go env GOPATH)/bin/air" ]; then \
		$$(go env GOPATH)/bin/air -c .air.control.toml; \
	else \
		echo "Air not found in PATH. Run 'make install-tools' or falling back to 'go run'..."; \
		go run ./cmd/mcrflow-control; \
	fi

dev-backend: dev-control

# Run Edge Agent with Air live reload
dev-agent:
	@echo "--> Starting MCRFlow Edge Playout Agent with Air live reload..."
	@if [ -x "$$(which air 2>/dev/null)" ]; then \
		air -c .air.agent.toml; \
	elif [ -x "$$(go env GOPATH)/bin/air" ]; then \
		$$(go env GOPATH)/bin/air -c .air.agent.toml; \
	else \
		echo "Air not found in PATH. Run 'make install-tools' or falling back to 'go run'..."; \
		go run ./cmd/mcrflow-agent; \
	fi

# Run UI standalone dev server on port 3000
dev-ui:
	@echo "--> Starting Web UI Dev Server on http://localhost:3000..."
	@if command -v npx >/dev/null 2>&1; then \
		npx serve ui-mockup -l 3000; \
	elif command -v python3 >/dev/null 2>&1; then \
		python3 -m http.server 3000 --directory ui-mockup; \
	elif command -v python >/dev/null 2>&1; then \
		python -m http.server 3000 --directory ui-mockup; \
	else \
		echo "Neither npx nor python found. Open ui-mockup/index.html in browser directly."; \
	fi

# Build all binaries
build: build-control build-agent
	@echo "--> All binaries successfully compiled to $(BIN_DIR)/"

build-control:
	@mkdir -p $(BIN_DIR)
	@echo "--> Building $(BIN_DIR)/mcrflow-control..."
	go build -v -o $(BIN_DIR)/mcrflow-control ./cmd/mcrflow-control

build-agent:
	@mkdir -p $(BIN_DIR)
	@echo "--> Building $(BIN_DIR)/mcrflow-agent..."
	go build -v -o $(BIN_DIR)/mcrflow-agent ./cmd/mcrflow-agent

# Testing targets
test:
	@echo "--> Running all tests..."
	go test -v ./...

test-unit:
	@echo "--> Running unit tests..."
	go test -v ./internal/...

test-e2e:
	@echo "--> Running end-to-end integration tests..."
	go test -v ./tests/e2e/...

test-race:
	@echo "--> Running tests with race detector..."
	go test -race -v ./...

# Code quality
fmt:
	@echo "--> Formatting Go code..."
	go fmt ./...

vet:
	@echo "--> Running go vet..."
	go vet ./...

tidy:
	@echo "--> Tidying Go dependencies..."
	go mod tidy

# Cleanup
clean:
	@echo "--> Cleaning build artifacts..."
	@rm -rf $(BIN_DIR) $(TMP_DIR) build-errors.log
	@echo "--> Clean complete."
