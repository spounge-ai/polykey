BIN_DIR = bin
SERVER_NAME = polykey
SERVER_MAIN_PKG = ./cmd/polykey
CLIENT_MAIN_PKG = ./cmd/dev_client

all: build-server

build-server: $(BIN_DIR)/$(SERVER_NAME)

$(BIN_DIR)/$(SERVER_NAME): $(shell find $(SERVER_MAIN_PKG) -name '*.go')
	@echo "Building $(SERVER_NAME)..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/$(SERVER_NAME) $(SERVER_MAIN_PKG)
	@echo "Build successful. Executable created at $(BIN_DIR)/$(SERVER_NAME)"

run-server: build-server
	@echo "Running $(SERVER_NAME)..."
	@$(BIN_DIR)/$(SERVER_NAME)

run-dev-client:
	@echo "Running dev_client..."
	@go run $(CLIENT_MAIN_PKG)/main.go
	@echo "dev_client finished."

clean:
	@echo "Cleaning up..."
	@rm -rf $(BIN_DIR)
	@echo "Clean complete."

.PHONY: all build-server run-server run-dev-client clean
