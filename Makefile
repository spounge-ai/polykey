.DEFAULT_GOAL := help
MAKEFLAGS += --no-print-directory

# Variables
BIN_DIR := bin
SERVER_BINARY := $(BIN_DIR)/polykey
CLIENT_BINARY := $(BIN_DIR)/dev_client
PORT := 50053

# Colors
CYAN := 
YELLOW := 
GREEN := 
RESET := 

.PHONY: build server client test clean kill help

build: ## Build binaries
	@mkdir -p $(BIN_DIR)
	@go build -ldflags="-s -w" -o $(SERVER_BINARY) ./cmd/polykey
	@go build -ldflags="-s -w" -o $(CLIENT_BINARY) ./cmd/dev_client

server: kill build ## Run server
	@echo "$(GREEN)Starting server on port $(PORT)...$(RESET)"
	@POLYKEY_CONFIG_PATH=configs/config.test.yaml POLYKEY_GRPC_PORT=$(PORT) $(SERVER_BINARY) &\
		echo $$! > .server_pid

client: build ## Run client
	@if ! nc -z localhost $(PORT) 2>/dev/null; then \
		echo "$(YELLOW)Server not running, starting...$(RESET)"; \
		$(MAKE) server; sleep 2; \
	fi
	@POLYKEY_GRPC_PORT=$(PORT) $(CLIENT_BINARY)

test: ## Run tests
	@go test -v ./...

test-race: ## Run tests with race detector
	@go test -race -v ./...

test-integration:
	@echo "Running integration tests..."
	@POLYKEY_CONFIG_PATH=./configs/config.local.yaml go test -tags=local_mocks -v ./tests/integration/...

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