# Polykey gRPC Client

A modern Go gRPC client for the Polykey service with production-ready features.

## Features

- **Modern gRPC**: Uses `grpc.NewClient()` with proper connection management
- **Graceful Shutdown**: Signal handling with context cancellation
- **Structured Logging**: JSON format with consistent field naming
- **Error Handling**: Wrapped errors with proper gRPC status codes
- **Connection Management**: Automatic connection state monitoring
- **Resource Cleanup**: Proper defer patterns for guaranteed cleanup

## Usage

```bash
go run main.go
```

## Configuration

The client loads configuration via `config.NewConfigLoader()` which should provide:
- `ServerAddress`: gRPC server endpoint
- `Timeout`: Connection timeout duration

## Architecture

### Client Structure
- `Client`: Wraps gRPC connection and service client
- `NewClient()`: Creates client with connection management
- `ExecuteTool()`: Executes tool requests with proper error handling

### Connection Management
- Uses modern `grpc.NewClient()` instead of deprecated `grpc.Dial()`
- Implements connection state monitoring
- Handles timeouts via context
- Automatic reconnection on idle state

### Error Handling
- Structured gRPC error logging
- Error wrapping with context
- Proper status code extraction
- Request-specific timeout handling

## Dependencies

```go
import (
    "google.golang.org/grpc"
    "google.golang.org/grpc/connectivity"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/keepalive"
    "google.golang.org/grpc/status"
)
```

## gRPC Configuration

- **Transport**: Insecure credentials (development only)
- **Keepalive**: 10s interval, 5s timeout
- **Message Size**: 4MB max send/receive
- **Connection Timeout**: Configurable via `cfg.Timeout`

## Logging

Uses structured JSON logging with fields:
- `tool_name`: Tool being executed
- `user_id`: User identifier
- `workflow_run_id`: Workflow run identifier
- `status_code`: Response status
- `error`: Error details

## Graceful Shutdown

Handles `SIGINT` and `SIGTERM` signals:
1. Context cancellation propagates to all operations
2. gRPC connection closes cleanly
3. Resources are properly released

## Testing

The client includes a test request to verify connectivity and functionality:
- Creates sample parameters
- Executes `ExecuteTool` RPC
- Logs response based on output type (string, struct, file)

## Production Considerations

- Replace `insecure.NewCredentials()` with TLS credentials
- Configure appropriate timeouts for your use case
- Monitor connection state for health checks
- Implement retry logic for transient failures
- Add metrics collection for observability