.DEFAULT_GOAL := help

# ==============================================================================
# VARIABLES
# ==============================================================================
# Binaries
BIN_DIR 				:= bin
SERVER_BINARY 			:= $(BIN_DIR)/polykey
CLIENT_BINARY 			:= $(BIN_DIR)/dev_client

# Go
GO 						:= go
GO_BUILD_FLAGS 			:= -a -installsuffix cgo
# LDFLAGS: -s strips debugging symbols, -w strips DWARF information. Reduces binary size and makes it harder to reverse-engineer.
LDFLAGS 				:= -ldflags="-s -w"
CGO_ENABLED 			:= CGO_ENABLED=0
GOOS 					:= GOOS=linux

# Docker & Compose
COMPOSE_FILE 			:= compose.yml
DOCKER_CMD 				:= docker compose -f $(COMPOSE_FILE)
SERVER_ADDR 			:= localhost:50051


# ==============================================================================
# COMMANDS
# ==============================================================================

.PHONY: all build build-server build-client run run-server run-client test test-race test-integration compose-up compose-down compose-dev compose-logs clean prune help help-setup

all: build ## Build both server and client binaries

# ------------------------------------------------------------------------------
# Build Commands
# ------------------------------------------------------------------------------
build: build-server build-client ## Build both server and client binaries

build-server: ## Build the server binary for a Linux environment
	@echo "--> Building server..."
	@mkdir -p $(BIN_DIR)
	$(CGO_ENABLED) $(GOOS) $(GO) build $(GO_BUILD_FLAGS) $(LDFLAGS) -o $(SERVER_BINARY) ./cmd/polykey

build-client: ## Build the client binary for a Linux environment
	@echo "--> Building client..."
	@mkdir -p $(BIN_DIR)
	$(CGO_ENABLED) $(GOOS) $(GO) build $(GO_BUILD_FLAGS) $(LDFLAGS) -o $(CLIENT_BINARY) ./cmd/dev_client

# ------------------------------------------------------------------------------
# Local Run Commands
# ------------------------------------------------------------------------------
run-server: ## Run the server locally using 'go run'
	@echo "--> Starting server locally..."
	@$(GO) run ./cmd/polykey

run-client: ## Run the client locally, targeting localhost
	@echo "--> Starting client locally..."
	@POLYKEY_SERVER_ADDR=$(SERVER_ADDR) $(GO) run ./cmd/dev_client

# ------------------------------------------------------------------------------
# Testing Commands
# ------------------------------------------------------------------------------
test: ## Run all unit tests
	@echo "--> Running unit tests..."
	@$(GO) test ./...

test-race: ## Run all unit tests with the race detector
	@echo "--> Running unit tests with race detector..."
	@$(GO) test -race ./...

test-integration: compose-up ## Run integration tests against the Docker environment
	@echo "--> Running integration tests..."
	@echo "Waiting for server to be healthy..."
	@sleep 5
	@POLYKEY_SERVER_ADDR=$(SERVER_ADDR) $(GO) test -v -tags=integration ./...
	@$(MAKE) compose-down

# ------------------------------------------------------------------------------
# Docker Compose Commands
# ------------------------------------------------------------------------------
compose-dev: ## Build and run the full dev environment (server & client)
	@echo "--> Starting development environment with Docker Compose..."
	@$(DOCKER_CMD) --profile dev up --build

compose-up: ## Build and run only the server via Docker Compose
	@echo "--> Starting server with Docker Compose..."
	@$(DOCKER_CMD) up --build -d polykey-server

compose-down: ## Stop and remove all Docker Compose containers
	@echo "--> Stopping Docker Compose environment..."
	@$(DOCKER_CMD) down

compose-logs: ## View logs from all running containers
	@echo "--> Tailing logs..."
	@$(DOCKER_CMD) logs -f

# ------------------------------------------------------------------------------
# Cleaning Commands
# ------------------------------------------------------------------------------
clean: ## Clean local build artifacts
	@echo "--> Cleaning local binaries..."
	@rm -rf $(BIN_DIR)

prune: compose-down ## Stop containers AND remove volumes and images
	@echo "--> Pruning Docker resources..."
	@$(DOCKER_CMD) down -v --rmi all --remove-orphans
	@docker image prune -f

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
