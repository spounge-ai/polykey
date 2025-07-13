.PHONY: all build-server run-server run-dev-client docker-build docker-run docker-push clean help

BIN_DIR := bin
SERVER_NAME := polykey
SERVER_MAIN_PKG := ./cmd/polykey
CLIENT_MAIN_PKG := ./cmd/dev_client
DOCKER_IMAGE := ghcr.io/$(shell echo ${GITHUB_REPOSITORY} | tr '[:upper:]' '[:lower:]'):latest
GO := go
PWD := $(shell pwd)

HOST_PORT ?= 50052
CONTAINER_PORT ?= 50051

all: build-server

build-server: $(BIN_DIR)/$(SERVER_NAME)

$(BIN_DIR)/$(SERVER_NAME): $(shell find $(SERVER_MAIN_PKG) -name '*.go')
	@echo "Building $(SERVER_NAME)..."
	@mkdir -p $(BIN_DIR)
	@$(GO) build -o $(BIN_DIR)/$(SERVER_NAME) $(SERVER_MAIN_PKG)
	@echo "Build complete: $(BIN_DIR)/$(SERVER_NAME)"

run-server:
	@echo "Running $(SERVER_NAME)..."
	@$(BIN_DIR)/$(SERVER_NAME)

run-dev-client:
	@echo "Running dev client..."
	@$(GO) run $(CLIENT_MAIN_PKG)/main.go
	@echo "Dev client finished."

docker-build:
	@echo "Building Docker image..."
	docker build \
		--pull \
		--cache-from=type=local,src=/tmp/.buildx-cache \
		--cache-to=type=local,dest=/tmp/.buildx-cache \
		-t polykey-server .

docker-run: docker-build
	@echo "Running Docker container (host port: $(HOST_PORT) → container port: $(CONTAINER_PORT))..."
	docker run --rm -p $(HOST_PORT):$(CONTAINER_PORT) polykey-server

docker-push: docker-build
	@echo "Pushing Docker image to registry..."
	docker push $(DOCKER_IMAGE)
	@echo "Image pushed: $(DOCKER_IMAGE)"

clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)
	@echo "Clean complete."

help:
	@echo "Makefile commands:"
	@echo "  all               - Build the polykey server binary (default)"
	@echo "  build-server      - Build the server binary into $(BIN_DIR)/$(SERVER_NAME)"
	@echo "  run-server        - Run the server binary locally"
	@echo "  run-dev-client    - Run the dev client locally"
	@echo "  docker-build      - Build the Docker image for the server"
	@echo "  docker-run        - Run the server in a Docker container"
	@echo "  docker-push       - Push the server image to GHCR"
	@echo "  clean             - Remove build artifacts"
	@echo "  help              - Show this help message"
