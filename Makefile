.DEFAULT_GOAL := help
MAKEFLAGS += --no-print-directory

# Variables
BIN_DIR := bin
SERVER_BINARY := $(BIN_DIR)/polykey
CLIENT_BINARY := $(BIN_DIR)/dev_client
PORT := 50053

# Configs
CONFIG_DIR := configs
CONFIG_TEST := $(CONFIG_DIR)/config.test.yaml
CONFIG_PROD := $(CONFIG_DIR)/config.production.yaml
CONFIG_MINIMAL := $(CONFIG_DIR)/config.minimal.yaml

# Colors
CYAN := 
YELLOW := 
GREEN := 
RESET := 

.PHONY: build build-test server server-test server-prod server-minimal client test clean kill help

build: ## Build production binaries
	@mkdir -p $(BIN_DIR)
	@go build -ldflags="-s -w" -o $(SERVER_BINARY) ./cmd/polykey
	@go build -ldflags="-s -w" -o $(CLIENT_BINARY) ./cmd/dev_client

build-test: ## Build binaries with mock dependencies for testing
	@mkdir -p $(BIN_DIR)
	@go build -ldflags="-s -w" -tags=local_mocks -o $(SERVER_BINARY) ./cmd/polykey
	@go build -ldflags="-s -w" -o $(CLIENT_BINARY) ./cmd/dev_client

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


client: build ## Run client

	@if ! nc -z localhost $(PORT) 2>/dev/null; then \
		echo "$(YELLOW)Server not running, starting...$(RESET)"; \
		$(MAKE) server; sleep 2; \
	fi
	@POLYKEY_GRPC_PORT=$(PORT) $(CLIENT_BINARY)

test: ## Run tests
	@go test -v -json ./... | tparse -all

test-race: ## Run tests with race detector
	@go test -race -v -json ./... | tparse -all

test-integration:
	@echo "Running integration tests..."
	@POLYKEY_CONFIG_PATH=./configs/config.local.yaml go test -tags=local_mocks -v -json ./tests/integration/... | tparse -all

clean: kill ## Clean build artifacts
	@rm -rf $(BIN_DIR) .server_pid server.log

kill: ## Kill server processes
	@if [ -f .server_pid ]; then \
		kill $$(cat .server_pid) >/dev/null 2>&1 || true; \
		rm -f .server_pid; \
	fi
	@-lsof -ti:$(PORT) | xargs kill -9 >/dev/null 2>&1 || true

help: ## Show help
	@echo "$(CYAN)Polykey Development$(RESET)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-15s$(RESET) %s\n", $$1, $$2}'