# Command line interface for the application.
.DEFAULT_GOAL := help
MAKEFLAGS += --no-print-directory

# ============================================================================
# Variables
# ============================================================================
# Binaries and Directories
BIN_DIR       := bin
SERVER_BINARY := $(BIN_DIR)/polykey
CLIENT_BINARY := $(BIN_DIR)/dev_client

# Go Build Configuration
LDFLAGS       := -ldflags="-s -w"
# Allow overriding build tags, e.g., `make build BUILD_TAGS=local_mocks`
BUILD_TAGS    ?=

# Server Configuration
PORT          := 50053
PK_ENV        := POLYKEY_GRPC_PORT=$(PORT)

# --- CONFIGURATION SELECTION ---
# This variable controls which configuration is used for development tasks
# like `make server`, `make migrate`, and `make test-integration`.
# It can be 'minimal', 'test', or 'production'.
# Override from the command line: `make server CONFIG_NAME=test`
CONFIG_NAME ?= minimal
# ---

# Configuration Files and Derived Settings
CONFIG_DIR        := configs
# Dynamically set the config file path based on CONFIG_NAME
CONFIG_FILE       := $(CONFIG_DIR)/config.$(CONFIG_NAME).yaml
# Use 'local_mocks' build tag automatically if the config is 'test'
SERVER_BUILD_TAGS := $(if $(filter test,$(CONFIG_NAME)),local_mocks)

# Test Configuration
# Enable race detector with `make test race=true`
ifeq ($(race),true)
    RACE_FLAG := -race
endif

# Colors
CYAN   := \033[36m
YELLOW := \033[33m
GREEN  := \033[32m
RESET  := \033[0m

# ============================================================================
# Phony Targets
# ============================================================================
.PHONY: all build clean test coverage lint server server-test server-prod server-minimal client client-debug migrate kill help init client-setup client-server test-integration test-persistence test-cockroachdb vuln-check sbom

# ============================================================================
# Core Targets
# ============================================================================
all: build

init: ## Initialize the project and make scripts executable
	@echo "$(CYAN)Initializing project...$(RESET)"
	@find scripts -type f \( -name "*.sh" -o -name "*.go" \) -exec chmod +x {} \;
	@echo "$(GREEN)Project initialized!$(RESET)"

lint: ## Run the Go linter
	@echo "$(CYAN)Running linter...$(RESET)"
	@golangci-lint run

build: ## Build binaries (use BUILD_TAGS=local_mocks for test build)
	@echo "$(CYAN)Building binaries (Tags: $(if $(BUILD_TAGS),$(BUILD_TAGS),'none'))...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@go build $(LDFLAGS) $(if $(BUILD_TAGS),-tags=$(BUILD_TAGS)) -o $(SERVER_BINARY) ./cmd/polykey
	@go build $(LDFLAGS) -o $(CLIENT_BINARY) ./cmd/dev_client
	@echo "$(GREEN)Build complete!$(RESET)"

# ============================================================================
# Server Targets
# ============================================================================
server: kill ## Run server with the config specified by CONFIG_NAME (default: minimal)
	@echo "$(CYAN)Building server for config profile: '$(CONFIG_NAME)'...$(RESET)"
	@$(MAKE) --silent build BUILD_TAGS="$(SERVER_BUILD_TAGS)"
	@echo "$(GREEN)Starting server with config '$(CONFIG_FILE)' on port $(PORT)...$(RESET)"
	@POLYKEY_CONFIG_PATH=$(CONFIG_FILE) $(PK_ENV) $(SERVER_BINARY) &

server-test: ## Alias for 'make server CONFIG_NAME=test'
	@$(MAKE) server CONFIG_NAME=test

server-prod: ## Alias for 'make server CONFIG_NAME=production'
	@$(MAKE) server CONFIG_NAME=production

server-minimal: ## Alias for 'make server CONFIG_NAME=minimal'
	@$(MAKE) server CONFIG_NAME=minimal

# ============================================================================
# Client Targets
# ============================================================================
client-setup: ## Setup the development client
	@echo "$(CYAN)Setting up development client...$(RESET)"
	@./scripts/setup-dev-client.sh
	@echo "$(GREEN)Client setup complete!$(RESET)"

client: build ## Run client (depends on a running server)
	@if ! nc -z localhost $(PORT) 2>/dev/null; then \
		echo "$(YELLOW)Server not running, please start it first (e.g., 'make server')$(RESET)"; \
		exit 1; \
	fi
	@echo "$(CYAN)Starting client...$(RESET)"
	@$(PK_ENV) $(CLIENT_BINARY)

client-debug: ## Run the development client with debugging enabled
	@echo "$(CYAN)Starting client with debugging...$(RESET)"
	@POLYKEY_DEBUG=true go run cmd/dev_client/main.go

client-server: lint ## Build, run server, wait, and run client (uses CONFIG_NAME)
	@$(MAKE) --silent server
	@echo "$(CYAN)Waiting for server to be ready on port $(PORT)...$(RESET)"; \
	timeout=10; \
	while ! nc -z localhost $(PORT) >/dev/null 2>&1; do \
		timeout=$$((timeout-1)); \
		if [ $$timeout -eq 0 ]; then \
			echo "$(YELLOW)Error: Server failed to start within 10 seconds.$(RESET)"; \
			exit 1; \
		fi; \
		sleep 1; \
	done; \
	echo "$(GREEN)Server is ready! Starting client...$(RESET)";
	@$(MAKE) --silent client

# ============================================================================
# Test & Coverage Targets
# ============================================================================
test: ## Run tests (use 'race=true' to enable the race detector)
	@echo "$(CYAN)Running tests... $(if $(RACE_FLAG),(with race detector))$(RESET)"
	@go test $(RACE_FLAG) -v -json ./... | tparse -all

test-race: ## Alias for 'make test race=true'
	@$(MAKE) test race=true

# Note: All subsequent test targets will use the config specified by CONFIG_NAME
test-integration:
	@echo "$(CYAN)Running integration tests with gotestsum...$(RESET)"
	@POLYKEY_CONFIG_PATH=$(abspath $(CONFIG_FILE)) gotestsum --format=testname -- ./tests/integration/...

test-persistence: ## Run persistence tests
	@echo "$(CYAN)Running persistence tests with config '$(CONFIG_FILE)'...$(RESET)"
	@POLYKEY_CONFIG_PATH=$(abspath $(CONFIG_FILE)) go test -v ./internal/infra/persistence/...

test-cockroachdb: ## Run CockroachDB tests
	@echo "$(CYAN)Running CockroachDB persistence tests with config '$(CONFIG_FILE)'...$(RESET)"
	@POLYKEY_CONFIG_PATH=$(abspath $(CONFIG_FILE)) go test -v ./tests/integration/persistence_cockroachdb_test.go

coverage: ## Generate and display test coverage report
	@echo "$(CYAN)Generating coverage report...$(RESET)"
	@go test -coverprofile=coverage.out ./...
	@echo "$(GREEN)Opening coverage report in browser...$(RESET)"
	@go tool cover -html=coverage.out

# ============================================================================
# Utility Targets
# ============================================================================
migrate: ## Run database migrations
	@echo "$(CYAN)Running database migrations with config '$(CONFIG_FILE)'...$(RESET)"
	@POLYKEY_CONFIG_PATH=$(CONFIG_FILE) go run cmd/utils/migrate.go

vuln-check: ## Run vulnerability check
	@echo "$(CYAN)Running vulnerability check...$(RESET)"
	@./scripts/vulncheck.sh

sbom: ## Generate SBOM
	@echo "$(CYAN)Generating SBOM...$(RESET)"
	@./scripts/generate_sbom.sh

# ============================================================================
# Cleanup Targets
# ============================================================================
clean: kill ## Clean build artifacts and logs
	@echo "$(YELLOW)Cleaning build artifacts...$(RESET)"
	@rm -rf $(BIN_DIR) .server_pid server.log coverage.out tests/integration.test
	@echo "$(GREEN)Cleanup complete!$(RESET)"

kill: ## Kill any running server processes on the configured port
	@echo "$(CYAN)Stopping server process on port $(PORT)...$(RESET)"
	@-lsof -ti:$(PORT) | xargs kill -9 >/dev/null 2>&1 || true

# ============================================================================
# Help Target
# ============================================================================
help: ## Show this help message
	@echo "$(CYAN)╔═══════════════════════════════════════════════════════════╗$(RESET)"
	@echo "$(CYAN)║                 Polykey Development                 ║$(RESET)"
	@echo "$(CYAN)╚═══════════════════════════════════════════════════════════╝$(RESET)"
	@echo ""
	@echo "$(YELLOW)Default config for dev/test is: $(CYAN)$(CONFIG_NAME) ($(CONFIG_FILE))$(RESET)"
	@echo "$(YELLOW)To override, use 'make <target> CONFIG_NAME=<name>', e.g., 'make server CONFIG_NAME=test'$(RESET)"
	@echo ""
	@echo "$(YELLOW)Available targets:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-18s$(RESET) %s\n", $$1, $$2}'
	@echo ""