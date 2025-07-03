# Define the output directory for executables
BIN_DIR = bin

# Define the name of the server executable
SERVER_NAME = polykey

# Define the path to the main package of the server
SERVER_MAIN_PKG = ./cmd/polykey

# Define the path to the main package of the dev client
CLIENT_MAIN_PKG = ./cmd/dev_client

# Default target: build the server
all: build-server

# Target to build the server executable
build-server: $(BIN_DIR)/$(SERVER_NAME)

$(BIN_DIR)/$(SERVER_NAME): $(SERVER_MAIN_PKG)/*.go
	@echo "Building $(SERVER_NAME)..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/$(SERVER_NAME) $(SERVER_MAIN_PKG)
	@echo "Build successful. Executable created at $(BIN_DIR)/$(SERVER_NAME)"

# Target to run the server executable
run-server: build-server
	@echo "Running $(SERVER_NAME)..."w
	@$(BIN_DIR)/$(SERVER_NAME)

# Target to run the dev client
run-dev-client:
	@echo "Running dev_client..."
	@go run $(CLIENT_MAIN_PKG)/main.go
	@echo "dev_client finished."

# Target to clean up built files
clean:
	@echo "Cleaning up..."
	@rm -rf $(BIN_DIR)
	@echo "Clean complete."

.PHONY: all build-server run-server clean run-dev-client