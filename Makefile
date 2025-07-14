.DEFAULT_GOAL := help

# ==============================================================================
# VARIABLES
# ==============================================================================
# Binaries
BIN_DIR                 := bin
SERVER_BINARY           := $(BIN_DIR)/polykey
CLIENT_BINARY           := $(BIN_DIR)/dev_client

# Go
GO                      := go
# Flags for final, cross-compiled production builds (slower)
GO_BUILD_FLAGS_PROD     := -a -installsuffix cgo -ldflags="-s -w"
# Flags for fast, local development builds (uses cache)
GO_BUILD_FLAGS_LOCAL    := -ldflags="-s -w"
CGO_ENABLED             := CGO_ENABLED=0
GOOS                    := GOOS=linux

# Docker & Compose
COMPOSE_FILE            := compose.yml
DOCKER_CMD              := docker compose -f $(COMPOSE_FILE)
SERVER_ADDR             := localhost:50051

# Colors
GREEN                   := \033[0;32m
YELLOW                  := \033[0;33m
CYAN                    := \033[0;36m
RESET                   := \033[0m


# ==============================================================================
# COMMANDS
# ==============================================================================

.PHONY: all build build-local build-server build-client run run-server run-client test test-race test-integration compose-up compose-down compose-dev compose-logs clean-all clean-local docker-clean docker-prune help help-setup

all: build-local ## ✨ Build local development binaries

# ------------------------------------------------------------------------------
# Build Commands
# ------------------------------------------------------------------------------
build: ## 🏭 Build production-ready binaries for Linux (slow, full rebuild)
	@echo "$(YELLOW)▶ Building production server binary...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@$(CGO_ENABLED) $(GOOS) $(GO) build $(GO_BUILD_FLAGS_PROD) -o $(SERVER_BINARY) ./cmd/polykey
	@echo "$(YELLOW)▶ Building production client binary...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@$(CGO_ENABLED) $(GOOS) $(GO) build $(GO_BUILD_FLAGS_PROD) -o $(CLIENT_BINARY) ./cmd/dev_client

build-local: ## 🛠️  Build development binaries using cache (fast)
	@echo "$(YELLOW)▶ Building local server binary...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@$(GO) build $(GO_BUILD_FLAGS_LOCAL) -o $(SERVER_BINARY) ./cmd/polykey
	@echo "$(YELLOW)▶ Building local client binary...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@$(GO) build $(GO_BUILD_FLAGS_LOCAL) -o $(CLIENT_BINARY) ./cmd/dev_client

# ------------------------------------------------------------------------------
# Local Run Commands
# ------------------------------------------------------------------------------
run-server: ## 🚀 Run the server locally using 'go run'
	@echo "$(GREEN)▶ Running server locally...$(RESET)"
	@$(GO) run ./cmd/polykey

run-test-client: ## 🚀 Run client with human-readable (text) logs
	@echo "$(GREEN)▶ Running client with @Meoya/Contour...$(RESET)"
	@LOG_FORMAT=text POLYKEY_SERVER_ADDR=$(SERVER_ADDR) $(GO) run ./cmd/dev_client

# ------------------------------------------------------------------------------
# Testing Commands
# ------------------------------------------------------------------------------
test: ## 🧪 Run unit tests and show a PASS/FAIL summary
	@echo "$(CYAN)▶ Running unit tests...$(RESET)"
	@$(GO) test -v -json ./... | tparse

test-race: ## 🧪 Run unit tests with the race detector and show a summary
	@echo "$(CYAN)▶ Running unit tests with race detector...$(RESET)"
	@$(GO) test -race -v -json ./... | tparse

test-integration: compose-up ## 🧪 Run integration tests and show a summary
	@echo "$(CYAN)▶ Running integration tests...$(RESET)"
	@echo "    (Waiting for server to become healthy)"
	@sleep 5
	@POLYKEY_SERVER_ADDR=$(SERVER_ADDR) $(GO) test -v -json -tags=integration ./... | tparse
	@$(MAKE) compose-down

# ------------------------------------------------------------------------------
# Docker Compose Commands
# ------------------------------------------------------------------------------
compose-dev: ## 🐳 Build and run the full dev environment (server & client)
	@echo "$(CYAN)▶ Starting full dev environment with Docker Compose...$(RESET)"
	@$(DOCKER_CMD) --profile dev up --build

compose-up: ## 🐳 Build and run only the server via Docker Compose
	@echo "$(CYAN)▶ Starting server with Docker Compose...$(RESET)"
	@$(DOCKER_CMD) up --build -d polykey-server

compose-down: ## 🐳 Stop and remove all Docker Compose containers
	@echo "$(YELLOW)▶ Stopping Docker Compose environment...$(RESET)"
	@$(DOCKER_CMD) down

compose-logs: ## 🐳 View logs from all running containers
	@echo "$(CYAN)▶ Tailing logs...$(RESET)"
	@$(DOCKER_CMD) logs -f

# ------------------------------------------------------------------------------
# Cleaning Commands
# ------------------------------------------------------------------------------
clean-all: clean-local docker-prune ## 🧹 Clean everything (local binaries and all Docker resources)

clean-local: ## 🧹 Clean local build artifacts only
	@echo "$(YELLOW)▶ Cleaning local binaries from ./bin...$(RESET)"
	@rm -rf $(BIN_DIR)

docker-clean: ## 🐳 Stop containers, remove networks and volumes
	@echo "$(YELLOW)▶ Cleaning project Docker containers, networks, and volumes...$(RESET)"
	@$(DOCKER_CMD) down -v --remove-orphans

docker-prune: ## 🐳 Clean everything Docker-related for the project, INCLUDING IMAGES
	@echo "$(YELLOW)▶ Pruning project Docker resources (containers, networks, volumes, and images)...$(RESET)"
	@$(DOCKER_CMD) down -v --rmi all --remove-orphans

# ------------------------------------------------------------------------------
# Help
# ------------------------------------------------------------------------------
help: ## ✨ Show this help message
	@echo "Usage: make [command]"
	@echo ""
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'


# ------------------------------------------------------------------------------
# Setup Help
# ------------------------------------------------------------------------------
help-setup: ## 📖 Explain the project's testing and running patterns
	@echo "\033[1;33mPolykey Service: How to Test and Run\033[0m"
	@echo ""
	@echo "\033[1;36m--- Testing Patterns ---\033[0m"
	@echo "1. \033[1;32mUnit Tests (Fast & Local):\033[0m"
	@echo "   Run quick checks on your local machine."
	@echo "   \033[35m> make test\033[0m or \033[35m> make test-race\033[0m"
	@echo ""
	@echo "2. \033[1;32mIntegration Tests (Full Stack):\033[0m"
	@echo "   Tests the full application using Docker. Slower but more thorough."
	@echo "   \033[35m> make test-integration\033[0m"
	@echo ""
	@echo "\033[1;36m--- Functional Run Patterns ---\033[0m"
	@echo "1. \033[1;32mRunning Locally (Go):\033[0m"
	@echo "   Ideal for quick, iterative development."
	@echo "   - In Terminal 1: \033[35m> make run-server\033[0m"
	@echo "   - In Terminal 2: \033[35m> make run-client\033[0m"
	@echo ""
	@echo "2. \033[1;32mRunning with Docker (Compose):\033[0m"
	@echo "   Runs the complete, containerized environment."
	@echo "   - To start everything: \033[35m> make compose-dev\033[0m"
	@echo "   - To stop everything:  \033[35m> make compose-down\033[0m"