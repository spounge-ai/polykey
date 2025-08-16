# Command line interface for the application.
.DEFAULT_GOAL := help
MAKEFLAGS += --no-print-directory

# ============================================================================ 
# Variables
# ============================================================================ 
BIN_DIR := bin
SERVER_BINARY := $(BIN_DIR)/polykey
CLIENT_BINARY := $(BIN_DIR)/dev_client
PORT := 50053

# ============================================================================ 
# Configuration Files
# ============================================================================ 
CONFIG_DIR := configs
CONFIG_TEST := $(CONFIG_DIR)/config.test.yaml
CONFIG_PROD := $(CONFIG_DIR)/config.production.yaml
CONFIG_MINIMAL := $(CONFIG_DIR)/config.minimal.yaml

# ============================================================================ 
# Colors
# ============================================================================ 
CYAN := \033[36m
YELLOW := \033[33m
GREEN := \033[32m
RESET := \033[0m

# ============================================================================ 
# Phony Targets
# ============================================================================ 
.PHONY: all build build-test clean test test-race test-integration coverage lint server server-test server-prod server-minimal client client-debug migrate kill help init

init: ## Initialize the project and make scripts executable
	@echo "$(CYAN)Initializing project...$(RESET)"
	@find scripts -type f -name "*.sh" -exec chmod +x {} \;
	@find scripts -type f -name "*.go" -exec chmod +x {} \;
	@echo "$(GREEN)Project initialized!$(RESET)"

all: build

lint:
	@echo "Running linter..."
	@golangci-lint run

client-debug:
	@echo "Starting client with debugging..."
	@POLYKEY_DEBUG=true go run cmd/dev_client/main.go

migrate:
	@echo "Running database migrations..."
	@POLYKEY_CONFIG_PATH=$(CONFIG_MINIMAL) go run cmd/utils/migrate.go

coverage:
	@echo "Displaying test coverage..."
	@go tool cover -html=coverage.out

# ============================================================================ 
# Build Targets
# ============================================================================ 

build: ## Build production binaries
	@mkdir -p $(BIN_DIR)
	@echo "$(CYAN)Building production binaries...$(RESET)"
	@go build -ldflags="-s -w" -o $(SERVER_BINARY) ./cmd/polykey
	@go build -ldflags="-s -w" -o $(CLIENT_BINARY) ./cmd/dev_client
	@echo "$(GREEN)Build complete!$(RESET)"

build-test: ## Build binaries with mock dependencies for testing
	@mkdir -p $(BIN_DIR)
	@echo "$(CYAN)Building test binaries with mocks...$(RESET)"
	@go build -ldflags="-s -w" -tags=local_mocks -o $(SERVER_BINARY) ./cmd/polykey
	@go build -ldflags="-s -w" -o $(CLIENT_BINARY) ./cmd/dev_client
	@echo "$(GREEN)Test build complete!$(RESET)"

# ============================================================================ 
# Server Targets
# ============================================================================ 

server: server-test ## Run server (defaults to test config)

server-test: kill build-test ## Run server with test config
	@echo "$(GREEN)Starting server with test config on port $(PORT)...$(RESET)"
	@POLYKEY_CONFIG_PATH=$(CONFIG_TEST) POLYKEY_GRPC_PORT=$(PORT) $(SERVER_BINARY) &

server-prod: kill build ## Run server with production config
	@echo "$(GREEN)Starting server with production config on port $(PORT)...$(RESET)"
	@POLYKEY_CONFIG_PATH=$(CONFIG_PROD) POLYKEY_GRPC_PORT=$(PORT) $(SERVER_BINARY) &

server-minimal: kill build ## Run server with minimal config
	@echo "$(GREEN)Starting server with minimal config on port $(PORT)...$(RESET)"
	@POLYKEY_CONFIG_PATH=$(CONFIG_MINIMAL) POLYKEY_GRPC_PORT=$(PORT) $(SERVER_BINARY) &

# ============================================================================ 
# Client Target
# ============================================================================ 

client: build ## Run client
	@if ! nc -z localhost $(PORT) 2>/dev/null; then \
		echo "$(YELLOW)Server not running, please start it with 'make server-minimal' or 'make server-test'$(RESET)"; \
		exit 1; \
	fi
	@echo "$(CYAN)Starting client...$(RESET)"
	@POLYKEY_GRPC_PORT=$(PORT) $(CLIENT_BINARY)

client-server: ## lint, run server-minimal, then client
	$(MAKE) lint
	$(MAKE) server-minimal
	$(MAKE) client

# ============================================================================ 
# Test Targets
# ============================================================================ 

test: ## Run tests
	@echo "$(CYAN)Running tests...$(RESET)"
	@go test -v -json ./... | tparse -all

test-race: ## Run tests with race detector
	@echo "$(CYAN)Running tests with race detector...$(RESET)"
	@go test -race -v -json ./... | tparse -all

test-integration: ## Run integration tests
	@echo "$(CYAN)Running integration tests...$(RESET)"
	@POLYKEY_CONFIG_PATH=../../configs/config.minimal.yaml go test -v ./tests/integration/...

test-persistence: ## Run persistence tests
	@echo "$(CYAN)Running persistence tests...$(RESET)"
	@POLYKEY_CONFIG_PATH=../../configs/config.minimal.yaml go test -v ./internal/infra/persistence/...

test-cockroachdb: ## Run CockroachDB persistence tests
	@echo "$(CYAN)Running CockroachDB persistence tests...$(RESET)"
	@POLYKEY_CONFIG_PATH=../../configs/config.minimal.yaml go test -v ./tests/integration/persistence_cockroachdb_test.go

vuln-check: ## Run vulnerability check
	@echo "$(CYAN)Running vulnerability check...$(RESET)"
	@./scripts/vulncheck.sh

sbom: ## Generate SBOM
	@echo "$(CYAN)Generating SBOM...$(RESET)"
	@./scripts/generate_sbom.sh

# ============================================================================ 
# Cleanup Targets
# ============================================================================ 

clean: kill ## Clean build artifacts
	@echo "$(YELLOW)Cleaning build artifacts...$(RESET)"
	@rm -rf $(BIN_DIR) .server_pid server.log
	@echo "$(GREEN)Cleanup complete!$(RESET)"

kill: ## Kill running server processes
	@if [ -f .server_pid ]; then \
		kill $$(cat .server_pid) >/dev/null 2>&1 || true; \
		rm -f .server_pid; \
	fi
	@-lsof -ti:$(PORT) | xargs kill -9 >/dev/null 2>&1 || true

# ============================================================================ 
# Help Target
# ============================================================================ 

help: ## Show help
	@echo "$(CYAN)╔═══════════════════════════════════════════════════════════╗$(RESET)"
	@echo "$(CYAN)║                    Polykey Development                    ║$(RESET)"
	@echo "$(CYAN)╚═══════════════════════════════════════════════════════════╝$(RESET)"
	@echo ""
	@echo "$(YELLOW)Available targets:$(RESET)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-18s$(RESET) %s\n", $$1, $$2}'
	@echo ""