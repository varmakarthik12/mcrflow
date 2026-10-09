# ==============================================================================
# MCRFlow Broadcast Automation - Cross-Platform Build & Development Pipeline
# ==============================================================================

# Variables for configurable service ports
UI ?= 3080
CONTROL ?= 3081
AGENT ?= 3082

BIN_DIR := ./bin
TMP_DIR := ./tmp
DATA_DIR := ./data
DIST_DIR := ./dist

# Cross-platform OS detection & shell configuration
ifeq ($(OS),Windows_NT)
  SHELL := cmd.exe
  .SHELLFLAGS := /c
  EXE := .exe
  MKDIR_BIN := if not exist bin mkdir bin
  CLEAN_CMD := if exist bin rmdir /s /q bin & if exist dist rmdir /s /q dist & if exist tmp rmdir /s /q tmp & if exist data rmdir /s /q data & if exist build-errors.log del /f /q build-errors.log
  AIR := $(shell where air 2>nul)
  SET_CONTROL_ENV := set MCRFLOW_PORT=$(CONTROL)&&
  SET_AGENT_ENV := set MCRFLOW_PORT=$(AGENT)&& set MCRFLOW_CONTROL_URL=http://localhost:$(CONTROL)&&
  SET_UI_ENV := set CONTROL=$(CONTROL)&& set MCRFLOW_CONTROL_URL=http://localhost:$(CONTROL)&&
else
  SHELL := /bin/bash
  .SHELLFLAGS := -c
  EXE :=
  MKDIR_BIN := mkdir -p $(BIN_DIR)
  CLEAN_CMD := rm -rf $(BIN_DIR) $(DIST_DIR) $(TMP_DIR) $(DATA_DIR) build-errors.log
  AIR := $(shell which air 2>/dev/null)
  SET_CONTROL_ENV := MCRFLOW_PORT=$(CONTROL)
  SET_AGENT_ENV := MCRFLOW_PORT=$(AGENT) MCRFLOW_CONTROL_URL=http://localhost:$(CONTROL)
  SET_UI_ENV := CONTROL=$(CONTROL) MCRFLOW_CONTROL_URL=http://localhost:$(CONTROL)
endif

.PHONY: help build build-control build-agent test test-unit test-e2e dev-control dev-agent dev-ui dev run clean docker-build docker-compose-up docker-compose-down fmt vet tidy

# Default target
help:
	@echo ================================================================================
	@echo   MCRFlow Developer Commands (Ports: UI=$(UI), Control=$(CONTROL), Agent=$(AGENT))
	@echo ================================================================================
	@echo   make build           Compile all binaries into $(BIN_DIR)/
	@echo   make build-control   Compile $(BIN_DIR)/mcrflow-control$(EXE)
	@echo   make build-agent     Compile $(BIN_DIR)/mcrflow-agent$(EXE)
	@echo   make test            Run all tests across the repository
	@echo   make dev-control     Run Control Plane on port $(CONTROL) (Air or go run)
	@echo   make dev-agent       Run Edge Playout Agent on port $(AGENT) (Air or go run)
	@echo   make dev-ui          Run Vite React Dev Server with HMR on port $(UI) (proxies to :$(CONTROL))
	@echo   make run             Run Control Plane service
	@echo   make docker-build    Build Docker images for Control Plane and Edge Agent
	@echo   make clean           Remove bin/, dist/, tmp/, data/ and logs
	@echo   make fmt             Format Go source files
	@echo   make vet             Run Go static analysis
	@echo   make tidy            Tidy Go module dependencies
	@echo ================================================================================

# Build all binaries
build: build-control build-agent
	@echo [MCRFlow] Build complete: binaries compiled in $(BIN_DIR)/

build-ui:
	@echo [MCRFlow] Building React Web UI...
	cd web && npm run build

build-control:
	@$(MKDIR_BIN)
	@echo [MCRFlow] Compiling $(BIN_DIR)/mcrflow-control$(EXE)...
	go build -v -o $(BIN_DIR)/mcrflow-control$(EXE) ./cmd/mcrflow-control

build-agent:
	@$(MKDIR_BIN)
	@echo [MCRFlow] Compiling $(BIN_DIR)/mcrflow-agent$(EXE)...
	go build -v -o $(BIN_DIR)/mcrflow-agent$(EXE) ./cmd/mcrflow-agent

# Run tests
test:
	@echo [MCRFlow] Running test suite...
	go test -v ./...

test-unit:
	@echo [MCRFlow] Running unit tests...
	go test -v ./internal/...

test-e2e:
	@echo [MCRFlow] Running end-to-end tests...
	go test -v ./tests/e2e/...

# Development live-reload targets
dev-control:
ifneq ($(strip $(AIR)),)
	@echo [MCRFlow] Starting Control Plane with Air live reload on port $(CONTROL)...
	$(SET_CONTROL_ENV) air -c .air.control.toml
else
	@echo [MCRFlow] Air not found; running Control Plane with go run on port $(CONTROL)...
	go run ./cmd/mcrflow-control -port $(CONTROL)
endif

dev-agent:
ifneq ($(strip $(AIR)),)
	@echo [MCRFlow] Starting Edge Playout Agent with Air live reload on port $(AGENT)...
	$(SET_AGENT_ENV) air -c .air.agent.toml
else
	@echo [MCRFlow] Air not found; running Edge Playout Agent with go run on port $(AGENT)...
	go run ./cmd/mcrflow-agent -port $(AGENT) -control-url http://localhost:$(CONTROL)
endif

dev-ui:
	@echo [MCRFlow] Starting Vite React Dev Server with HMR on port $(UI) (proxying to :$(CONTROL))...
	$(SET_UI_ENV) npm --prefix web run dev -- --port $(UI)

# Default development runner
run: dev-control
dev: dev-control

# Docker container targets
docker-build:
	@echo [MCRFlow] Building Docker image for varmakarthik12/mcrflow-control:latest...
	docker build -t varmakarthik12/mcrflow-control:latest -f docker/Dockerfile.control .
	@echo [MCRFlow] Building Docker image for varmakarthik12/mcrflow-agent:latest...
	docker build -t varmakarthik12/mcrflow-agent:latest -f docker/Dockerfile.agent .
	@echo [MCRFlow] Docker images successfully built.

docker-compose-up:
	@echo [MCRFlow] Launching MCRFlow stack via Docker Compose...
	docker compose -f docker/docker-compose.yml up -d

docker-compose-down:
	@echo [MCRFlow] Stopping MCRFlow stack...
	docker compose -f docker/docker-compose.yml down

# Cleanup targets
clean:
	@echo [MCRFlow] Cleaning build artifacts and data...
	@$(CLEAN_CMD)
	@echo [MCRFlow] Clean complete.

# Code hygiene
fmt:
	@echo [MCRFlow] Formatting Go code...
	go fmt ./...

vet:
	@echo [MCRFlow] Running go vet...
	go vet ./...

tidy:
	@echo [MCRFlow] Tidying dependencies...
	go mod tidy
