.DEFAULT_GOAL := help
MAKEFLAGS += --no-print-directory

# ==============================================================================
# PHONY TARGETS
# ==============================================================================
.PHONY: all build build-local ._build_binary run-server run-test-client \
		test test-race test-integration \
		compose-up compose-down compose-dev compose-logs compose-reboot \
		rebuild-docker-server rebuild-docker-server-force \
		clean-all clean-local docker-clean docker-prune \
		kill-local-server \
		help help-setup install-deps \
		security-scan security-scan-docker security-scan-fast security-clean-cache \
		security-scan-local-cache \
		build-docker-image docker-help fix-permissions ci-check

# ==============================================================================
# VARIABLES
# ==============================================================================
# Binaries
BIN_DIR 			 := bin
SERVER_BINARY 		 := $(BIN_DIR)/polykey
CLIENT_BINARY 		 := $(BIN_DIR)/dev_client

# Go
GO 					 := go
GO_BUILD_FLAGS_PROD  := -a -installsuffix cgo -ldflags="-s -w"
GO_BUILD_FLAGS_LOCAL := -ldflags="-s -w"
# CGO_ENABLED is passed as an environment variable to make for production builds
# E.g., CGO_ENABLED=0 make build
CGO_ENABLED_FLAG     ?= # Default to empty, allowing host's CGO_ENABLED

# Docker & Compose
COMPOSE_FILE 		 := compose.yml
DOCKER_CMD 			 := docker compose -f $(COMPOSE_FILE)
SERVER_ADDR 		 := localhost:50051
POLYKEY_SERVER_PORT  := 50051
service ?= # Variable for compose-logs to select a service

# Docker Image Tagging
DOCKER_IMAGE_NAME    := spoungeai/polykey-service
VERSION              ?= $(shell git rev-parse --short HEAD 2>/dev/null || date +%Y%m%d%H%M%S) # Git commit hash or timestamp
DOCKER_IMAGE_TAG     ?= latest # Default tag for build-docker-image
# Use 'make build-docker-image TAG=v1.0.0' to override DOCKER_IMAGE_TAG

# Colors
GREEN 				 := \033[0;32m
YELLOW 				 := \033[0;33m
CYAN 				 := \033[0;36m
RESET 				 := \033[0m

# ==============================================================================
# COMMANDS
# ==============================================================================

all: build-local ## ✨ Build local development binaries

# ------------------------------------------------------------------------------
# Build Commands
# ------------------------------------------------------------------------------
# Internal helper for building Go binaries
._build_binary:
	@mkdir -p $(BIN_DIR)
	@$(CGO_ENABLED_FLAG) $(GOOS) $(GO) build $(FLAGS) -o $(BINARY) ./cmd/$(CMD_NAME)

build: ## 🏭 Build production-ready binaries for Linux (slow, full rebuild)
	@echo "$(YELLOW)▶ Building production server binary...$(RESET)"
	@$(MAKE) ._build_binary FLAGS="$(GO_BUILD_FLAGS_PROD)" GOOS="GOOS=linux" BINARY="$(SERVER_BINARY)" CMD_NAME="polykey" CGO_ENABLED_FLAG="CGO_ENABLED=0"
	@echo "$(YELLOW)▶ Building production client binary...$(RESET)"
	@$(MAKE) ._build_binary FLAGS="$(GO_BUILD_FLAGS_PROD)" GOOS="GOOS=linux" BINARY="$(CLIENT_BINARY)" CMD_NAME="dev_client" CGO_ENABLED_FLAG="CGO_ENABLED=0"

build-local: ## 🛠️  Build development binaries using cache (fast)
	@echo "$(YELLOW)▶ Building local server binary...$(RESET)"
	@$(MAKE) ._build_binary FLAGS="$(GO_BUILD_FLAGS_LOCAL)" BINARY="$(SERVER_BINARY)" CMD_NAME="polykey"
	@echo "$(YELLOW)▶ Building local client binary...$(RESET)"
	@$(MAKE) ._build_binary FLAGS="$(GO_BUILD_FLAGS_LOCAL)" BINARY="$(CLIENT_BINARY)" CMD_NAME="dev_client"

build-production: ## 🏭 Build production-optimized Docker image (using 'production' target in Dockerfile)
	@echo "$(CYAN)▶ Building production Docker image: $(DOCKER_IMAGE_NAME):production-$(VERSION)...$(RESET)"
	# Changed from $(DOCKER_CMD) build to docker build
	@docker build --build-arg COMPRESS_BINARIES=true --target production --tag $(DOCKER_IMAGE_NAME):production-$(VERSION) .

build-docker-image: ## 🐳 Build and tag the Docker server image (e.g., 'make build-docker-image TAG=v1.0.0')
	@echo "$(CYAN)▶ Building Docker image: $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)...$(RESET)"
	# Changed from $(DOCKER_CMD) build to docker build
	@docker build --target server --tag $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG) .

# ------------------------------------------------------------------------------
# Local Run Commands (on the host machine)
# ------------------------------------------------------------------------------
run-server: ## 🚀 Run the server locally using 'go run'
	@echo "$(GREEN)▶ Running server locally...$(RESET)"
	@$(GO) run ./cmd/polykey

run-test-client: ## 🚀 Run client with human-readable (text) logs (host machine)
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

test-integration: compose-up ## 🧪 Run integration tests (waits for server to be healthy)
	@echo "$(CYAN)▶ Running integration tests...$(RESET)"
	@echo "    (Waiting for polykey-server to become healthy)"
	@until [ "$$(docker inspect -f {{.State.Health.Status}} $$(docker compose ps -q polykey-server))" = "healthy" ]; do \
		sleep 1; \
	done;
	@echo "$(GREEN)    Server is healthy! Running tests...$(RESET)"
	@POLYKEY_SERVER_ADDR=$(SERVER_ADDR) $(GO) test -v -json -tags=integration ./... | tparse
	@echo "$(GREEN)▶ Running dev client test...$(RESET)"
	@$(MAKE) run-test-client
	@$(MAKE) compose-down

# ------------------------------------------------------------------------------
# Docker Compose Commands
# ------------------------------------------------------------------------------
compose-dev: ## 🐳 Build and run the full dev environment (server & client). Waits for server health.
	@echo "$(CYAN)▶ Starting full dev environment (server & client)...$(RESET)"
	@$(DOCKER_CMD) up --build -d
	@echo "$(CYAN)    (Waiting for polykey-server to become healthy)"
	@until [ "$$(docker inspect -f {{.State.Health.Status}} $$(docker compose ps -q polykey-server))" = "healthy" ]; do \
		sleep 1; \
	done;
	@echo "$(GREEN)    Polykey server is healthy and ready!$(RESET)"

compose-up: ## 🐳 Build and run only the server for integration tests
	@echo "$(CYAN)▶ Starting server only...$(RESET)"
	@$(DOCKER_CMD) up --build -d polykey-server

compose-down: ## 🐳 Stop and remove all Docker Compose containers
	@echo "$(YELLOW)▶ Stopping Docker Compose environment...$(RESET)"
	@$(DOCKER_CMD) down --remove-orphans

compose-logs: ## 🐳 View logs from containers (e.g., 'make compose-logs s=polykey-server b=true')
	@echo "$(CYAN)▶ Tailing logs for: $(or $(s), 'all services')...$(RESET)"
	@if [ "$(b)" = "true" ]; then \
		echo "$(CYAN)    (Beautified output enabled. Using 'go run ./cmd/utils/log-beautifier')$(RESET)"; \
		$(DOCKER_CMD) logs -f $(s) | go run ./cmd/utils/log-beautifier; \
	else \
		$(DOCKER_CMD) logs -f $(s); \
	fi

# NOTE: This target assumes you have a 'polykey-dev-client' service defined in your compose.yml.
# If your client runs directly on the host (as was the successful case), use 'make run-test-client'
# after 'make compose-up' or 'make compose-dev' to interact with the Dockerized server.
compose-run-client: ## 📞 Run the dev-client task (requires server to be running via compose)
	@echo "$(GREEN)▶ Calling server with dev-client (Docker Compose service)...$(RESET)"
	@$(DOCKER_CMD) run --rm --no-deps polykey-dev-client

compose-reboot: ## ♻️ Reboot the server environment (down + up)
	@echo "$(YELLOW)▶ Rebooting Docker Compose environment...$(RESET)"
	@$(MAKE) compose-down
	@$(MAKE) compose-up

rebuild-docker-server: build-local ## 🔄 Rebuild Polykey server Docker image with latest local binaries and restart Compose
	@echo "$(CYAN)▶ Rebuilding Polykey server Docker image and restarting Compose services...$(RESET)"
	@$(MAKE) build-docker-image
	@$(MAKE) compose-reboot

rebuild-docker-server-force: build-local ## 💥 Force rebuild Polykey server Docker image (no cache) with latest local binaries and restart Compose
	@echo "$(CYAN)▶ FORCING complete rebuild of Polykey server Docker image (no cache) and restarting Compose services...$(RESET)"
	@$(DOCKER_CMD) build --no-cache polykey-server
	@$(DOCKER_CMD) restart polykey-server

# ------------------------------------------------------------------------------
# Cleaning Commands
# ------------------------------------------------------------------------------
clean-all: clean-local docker-prune ## 🧹 Clean everything (local binaries and all Docker resources)

clean-local: ## 🧹 Clean local build artifacts only
	@echo "$(YELLOW)▶ Cleaning local binaries from ./bin...$(RESET)"
	@rm -rf $(BIN_DIR)

docker-clean: ## 🐳 Stop containers and remove networks and volumes
	@echo "$(YELLOW)▶ Cleaning project containers, networks, and volumes...$(RESET)"
	@$(DOCKER_CMD) down -v --remove-orphans

docker-prune: ## ☠️  [DESTRUCTIVE] Clean everything, INCLUDING IMAGES. Asks for confirmation.
	@echo "$(YELLOW)WARNING: This will permanently delete all Docker images used by this project.$(RESET)"
	@printf "Are you sure you want to continue? [y/N] "; \
	read ans; \
	if [ "$$ans" = "y" ] || [ "$$ans" = "Y" ]; then \
		echo "▶ Pruning project Docker resources..."; \
		$(DOCKER_CMD) down -v --rmi all --remove-orphans; \
	else \
		echo "Prune operation cancelled."; \
	fi

kill-local-server: ## 🔪 Kill any processes listening on the local Polykey server port (50051)
	@echo "$(YELLOW)▶ Attempting to kill processes on port $(POLYKEY_SERVER_PORT)...$(RESET)"
	@if command -v lsof >/dev/null 2>&1; then \
		PIDS=$$(lsof -ti:$(POLYKEY_SERVER_PORT)); \
		if [ -n "$$PIDS" ]; then \
			echo "$(RED)  Found processes: $$PIDS. Killing them...$(RESET)"; \
			kill -9 $$PIDS; \
			echo "$(GREEN)  Processes on port $(POLYKEY_SERVER_PORT) killed.$(RESET)"; \
		else \
			echo "$(GREEN)  No processes found on port $(POLYKEY_SERVER_PORT).$(RESET)"; \
		fi; \
	else \
		echo "$(RED)  'lsof' command not found. Cannot kill processes by port. Please install lsof or kill manually.$(RESET)"; \
		echo "  (e.g., 'netstat -tulnp | grep :$(POLYKEY_SERVER_PORT)' to find PID, then 'kill -9 <PID>')"; \
	fi

# ------------------------------------------------------------------------------
# Security Scanning Commands (Requires Trivy to be installed)
# ------------------------------------------------------------------------------
security-scan: ## 🔍 Run security scan with local Trivy (fastest, requires install)
	@echo "$(CYAN)▶ Running security scan with local Trivy...$(RESET)"
	@if [ ! -d "bin" ]; then \
		echo "$(YELLOW)⚠️  bin/ directory not found. Building binaries first...$(RESET)"; \
		$(MAKE) build-local; \
	fi
	@if ! command -v trivy > /dev/null 2>&1; then \
		echo "$(YELLOW)⚠️  Trivy not found. Install with: make install-trivy$(RESET)"; \
		exit 1; \
	fi
	@trivy fs $(SERVER_BINARY) $(CLIENT_BINARY) # Scan the specific binaries

security-scan-docker: ## 🔍 Run security scan via Docker (with persistent cache in user's home)
	@echo "$(CYAN)▶ Running security scan via Docker (with cache)...$(RESET)"
	@if [ ! -d "bin" ]; then \
		echo "$(YELLOW)⚠️  bin/ directory not found. Building binaries first...$(RESET)"; \
		$(MAKE) build-local; \
	fi
	@mkdir -p $$HOME/.cache/trivy
	@docker run --rm \
		-v $(PWD):/workspace \
		-v $$HOME/.cache/trivy:/root/.cache/trivy:Z \
		-e TRIVY_CACHE_DIR=/root/.cache/trivy \
		aquasec/trivy fs /workspace/$(SERVER_BINARY) /workspace/$(CLIENT_BINARY)

security-scan-docker-volume: ## 🔍 Run security scan via Docker (with named volume for cache)
	@echo "$(CYAN)▶ Running security scan via Docker (named volume)...$(RESET)"
	@if [ ! -d "bin" ]; then \
		echo "$(YELLOW)⚠️  bin/ directory not found. Building binaries first...$(RESET)"; \
	fi
	@docker volume create trivy-cache 2>/dev/null || true
	@docker run --rm \
		-v $(PWD):/workspace \
		-v trivy-cache:/root/.cache/trivy \
		aquasec/trivy fs /workspace/$(SERVER_BINARY) /workspace/$(CLIENT_BINARY)

install-trivy: ## 📦 Install Trivy locally to ~/.local/bin
	@echo "$(GREEN)▶ Installing Trivy locally...$(RESET)"
	@mkdir -p ~/.local/bin
	@curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b ~/.local/bin
	@echo "$(GREEN)▶ Add ~/.local/bin to your PATH if not already there: export PATH=\"\$$HOME/.local/bin:\$$PATH\"$(RESET)"

security-clean-cache: ## 🧹 Clean Trivy caches (local and Docker volume) to save disk space
	@echo "$(YELLOW)▶ Cleaning Trivy cache...$(RESET)"
	@if [ -d "$$HOME/.cache/trivy" ]; then \
		echo "$(YELLOW)  Cleaning user cache (using Docker to handle permissions)...$(RESET)"; \
		docker run --rm -v $$HOME/.cache/trivy:/cache alpine rm -rf /cache/*; \
		rmdir $$HOME/.cache/trivy 2>/dev/null || true; \
	fi
	@if [ -d ".trivy-cache" ]; then \
		echo "$(YELLOW)  Cleaning .trivy-cache project directory...$(RESET)"; \
		docker run --rm -v $(PWD)/.trivy-cache:/cache alpine rm -rf /cache/*; \
		rmdir .trivy-cache 2>/dev/null || true; \
	fi
	@docker volume rm trivy-cache 2>/dev/null || true
	@echo "$(GREEN)▶ Trivy cache cleaned$(RESET)"

	
security-scan-local-cache: ## 🔍 Run security scan with local Trivy (with cache for CI)
	@echo "$(CYAN)▶ Running security scan with local Trivy (cached)...$(RESET)"
	@if [ ! -d "bin" ]; then \
		echo "$(YELLOW)⚠️ bin/ directory not found. Building binaries first...$(RESET)"; \
		$(MAKE) build-local; \
	fi
	@if ! command -v trivy > /dev/null 2>&1; then \
		echo "$(YELLOW)⚠️ Trivy not found. Falling back to Docker with cache...$(RESET)"; \
		$(MAKE) security-scan-docker; \
	else \
		echo "$(GREEN)▶ Using local Trivy installation$(RESET)"; \
		mkdir -p .trivy-cache; \
		TRIVY_CACHE_DIR=.trivy-cache trivy fs bin/ \
			--format table \
			--exit-code 1 \
			--ignore-unfixed \
			--vuln-type os,library \
			--severity CRITICAL,HIGH; \
	fi


ci-check: ## 🔍 Run all CI checks locally (build, lint, test, security scan)
	@echo "$(CYAN)▶ Running CI checks locally...$(RESET)"
	@echo "$(CYAN)▶ Building binaries first...$(RESET)"
	@$(MAKE) build-local
	@echo "$(CYAN)▶ Running linting...$(RESET)"
	@golangci-lint run
	@echo "$(CYAN)▶ Running unit tests...$(RESET)"
	@$(MAKE) test
	@echo "$(CYAN)▶ Running integration tests...$(RESET)"
	@$(MAKE) test-integration
	@echo "$(CYAN)▶ Running security scan...$(RESET)"
	@$(MAKE) security-scan-local-cache
	@echo "$(GREEN)✅ All CI checks passed!$(RESET)"

# ------------------------------------------------------------------------------
# Setup & Help
# ------------------------------------------------------------------------------
fix-permissions: ## 🔒 Fix permissions for generated files and caches
	@echo "$(CYAN)▶ Fixing permissions for generated files and caches...$(RESET)"
	@sudo chown -R $(shell id -u):$(shell id -g) .

install-deps: ## 📦 Install Go modules and development tools (tparse, grpc-health-probe)
	@echo "$(GREEN)▶ Downloading Go module dependencies...$(RESET)"
	@$(GO) mod tidy
	@echo "$(GREEN)▶ Installing development tools...$(RESET)"
	@$(GO) install github.com/mfridman/tparse@latest
	@$(GO) install github.com/grpc-ecosystem/grpc-health-probe@latest

help: ## ✨ Show this help message
	@echo "$(CYAN)========================================$(RESET)"
	@echo "$(GREEN)             Make Help Menu             $(RESET)"
	@echo "$(CYAN)========================================$(RESET)"
	@echo ""
	@echo "Usage: make [command]"
	@echo ""
	@echo "$(YELLOW)--- Build Commands ---$(RESET)"
	@grep -E '^(build|build-local|build-production|build-docker-image):.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}' | \
		sort
	@echo ""
	@echo "$(YELLOW)--- Local Run Commands ---$(RESET)"
	@grep -E '^(run-server|run-test-client):.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}' | \
		sort
	@echo ""
	@echo "$(YELLOW)--- Testing Commands ---$(RESET)"
	@grep -E '^(test|test-race|test-integration):.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}' | \
		sort
	@echo ""
	@echo "$(YELLOW)--- Docker Compose Commands ---$(RESET)"
	@grep -E '^(compose-up|compose-down|compose-dev|compose-logs|compose-run-client|compose-reboot|rebuild-docker-server|rebuild-docker-server-force):.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}' | \
		sort
	@echo ""
	@echo "$(YELLOW)--- Cleaning Commands ---$(RESET)"
	@grep -E '^(clean-all|clean-local|docker-clean|docker-prune|kill-local-server):.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}' | \
		sort
	@echo ""
	@echo "$(YELLOW)--- Security Scanning Commands ---$(RESET)"
	@grep -E '^(security-scan|security-scan-docker|security-scan-docker-volume|install-trivy|security-clean-cache|security-scan-local-cache):.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}' | \
		sort
	@echo ""
	@echo "$(YELLOW)--- CI/Permissions Commands ---$(RESET)"
	@grep -E '^(ci-check|fix-permissions):.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}' | \
		sort
	@echo ""
	@echo "$(YELLOW)--- Setup & Help Commands ---$(RESET)"
	@grep -E '^(install-deps|help|docker-help|help-setup):.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}' | \
		sort
	@echo "$(CYAN)========================================$(RESET)"

docker-help: ## 🐳 Show Docker-specific help and tagging information
	@echo "\033[1;33mDocker Commands & Tagging\033[0m"
	@echo ""
	@echo "This Makefile provides targets for managing Docker images and containers."
	@echo ""
	@echo "\033[1;36m--- Image Tagging ---\033[0m"
	@echo "The base Docker image name is: \033[35m$(DOCKER_IMAGE_NAME)\033[0m"
	@echo "Default tag for 'build-docker-image' is 'latest' or 'dev-<git_hash/timestamp>'."
	@echo "You can override the tag using the 'TAG' variable (e.g., 'make build-docker-image TAG=v1.0.0'):"
	@echo "  \033[35m> make build-docker-image TAG=v1.0.0\033[0m"
	@echo "  \033[35m> make build-docker-image TAG=custom-build-$(shell date +%Y%m%d)\033[0m"
	@echo ""
	@echo "\033[1;36m--- Docker Build Targets ---\033[0m"
	@echo " \033[36mbuild-docker-image\033[0m  - Builds the main Docker server image with a specified tag (or default)."
	@echo " \033[36mbuild-production\033[0m    - Builds the production-optimized Docker image."
	@echo ""
	@echo "\033[1;36m--- Docker Compose Targets ---\033[0m"
	@echo " \033[36mcompose-dev\033[0m        - Builds and runs the development environment via Docker Compose. Waits for server health."
	@echo " \033[36mcompose-up\033[0m         - Brings up Docker Compose services (server only by default)."
	@echo " \033[36mcompose-down\033[0m       - Stops and removes Docker Compose services."
	@echo " \033[36mcompose-reboot\033[0m     - Performs a 'down' followed by an 'up'."
	@echo " \033[36mrebuild-docker-server\033[0m - Rebuilds the server's Docker image with fresh local binaries and restarts services."
	@echo " \033[36mrebuild-docker-server-force\033[0m - Forces a complete rebuild of the server's Docker image (no cache) and restarts services."
	@echo " \033[36mcompose-logs\033[0m       - Tails logs from Docker Compose services. Use 's=<service_name>' and 'b=true' for beautified logs."
	@echo "    Example: \033[35m> make compose-logs s=polykey-server b=true\033[0m"
	@echo " \033[36mcompose-run-client\033[0m - Runs a client service defined in docker-compose.yml (if any)."
	@echo "    Note: Your host-based client (\033[35mrun-test-client\033[0m) can also connect to compose services."
	@echo ""
	@echo "\033[1;36m--- Docker Cleaning Targets ---\033[0m"
	@echo " \033[36mdocker-clean\033[0m       - Stops containers and removes networks/volumes (non-destructive to images)."
	@echo " \033[36mdocker-prune\033[0m       - \033[0;31m[DESTRUCTIVE]\033[0m Stops containers and removes ALL images and dangling resources. Asks for confirmation."
	@echo " \033[36mclean-all\033[0m        - Combines 'clean-local' and 'docker-prune'."
	@echo " \033[36mkill-local-server\033[0m  - Kills any processes listening on the Polykey server's local port (50051)."
	@echo ""

help-setup: ## 📖 Explain the project's testing and running patterns
	@echo "\033[1;33mPolykey Service: How to Test and Run\033[0m"
	@echo ""
	@echo "\033[1;36m--- Testing Patterns ---\033[0m"
	@echo "1. \033[1;32mUnit Tests (Fast & Local):\033[0m"
	@echo "   Run quick checks on your local machine."
	@echo "   \033[35m> make test\033[0m or \033[35m> make test-race\033[0m"
	@echo ""
	@echo "2. \033[1;32mIntegration Tests (Full Stack):\033[0m"
	@echo "   Tests the full application using Docker. Slower but more thorough."
	@echo "   \033[35m> make test-integration\033[0m"
	@echo ""
	@echo "\033[1;36m--- Functional Run Patterns ---\033[0m"
	@echo "1. \033[1;32mRunning Locally (Go):\033[0m"
	@echo "   Ideal for quick, iterative development."
	@echo "   - In Terminal 1: \033[35m> make run-server\033[0m"
	@echo "   - In Terminal 2: \033[35m> make run-test-client\033[0m"
	@echo ""
	@echo "2. \033[1;32mRunning with Docker (Compose):\033[0m"
	@echo "   Runs the complete, containerized environment."
	@echo "   - To start everything: \033[35m> make compose-dev\033[0m (waits for server to be healthy)"
	@echo "   - To stop everything:  \033[35m> make compose-down\033[0m"
	@echo "   - To run the host client against Docker: \033[35m> make run-test-client\033[0m (after compose-dev)"