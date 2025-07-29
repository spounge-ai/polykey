# PolyKey – Makefile Guide

This project uses a `Makefile` to manage development, testing, and container workflows.

## Commands

### Build

| Command            | Description                                   |
| ------------------ | --------------------------------------------- |
| `make build`       | Build production binaries for Linux.          |
| `make build-local` | Build development binaries for your local OS. |

### Run Locally (No Docker)

| Command                | Description                         |
| ---------------------- | ----------------------------------- |
| `make run-server`      | Run the gRPC server locally.        |
| `make run-test-client` | Run the development client locally. |

### Testing

| Command                 | Description                            |
| ----------------------- | -------------------------------------- |
| `make test`             | Run unit tests.                        |
| `make test-race`        | Run unit tests with the race detector. |
| `make test-integration` | Run full integration tests in Docker.  |

### Docker Compose

| Command                   | Description                                   |
| ------------------------- | --------------------------------------------- |
| `make compose-up`         | Start `polykey-server` and dependencies.      |
| `make compose-dev`        | Start full dev environment (server + client). |
| `make compose-run-client` | Run the client against the running server.    |
| `make compose-logs`       | Show logs from running containers.            |
| `make compose-down`       | Stop and remove containers and networks.      |

 
# Saved Info:

 PolyKey Server: Future Improvements
This document outlines recommended future enhancements to elevate the PolyKey gRPC server to a fully production-grade, scalable, and observable service.

1. Advanced Configuration Management
As the service grows, managing configuration via simple environment variables becomes insufficient. Adopting a dedicated configuration library is essential for operational flexibility.

Recommendation: Integrate Viper.

Benefits:

Unified Configuration: Manage settings from files (e.g., config.yaml), environment variables, and command-line flags in a single, prioritized system.

Dynamic Reloading: Allows for configuration changes without restarting the server.

Type-Safe Access: Reduces runtime errors by unmarshaling configuration into Go structs.

2. Enhanced Observability
To effectively monitor and debug the service in a production environment, we must move beyond basic logging.

2.1. Metrics for Monitoring and Alerting
Recommendation: Instrument the server to expose key performance indicators as Prometheus metrics.

Implementation:

Use a library like grpc-go-prometheus to automatically collect standard gRPC metrics (request latency, error counts, active connections).

Expose these metrics on a separate, internal-only HTTP port (e.g., :9090).

Benefits: Enables the creation of Grafana dashboards for real-time health monitoring and allows for sophisticated alerting rules based on service performance.

2.2. Distributed Tracing
In a microservices architecture, tracing a single request across multiple services is critical for debugging latency and errors.

Recommendation: Adopt OpenTelemetry for Go.

Implementation:

Add an OpenTelemetry interceptor to the gRPC server to automatically propagate trace contexts.

Export traces to a compatible backend like Jaeger or Honeycomb.

Benefits: Provides a complete, end-to-end visualization of request lifecycles, making it invaluable for identifying performance bottlenecks.

3. Performance Profiling
For diagnosing high CPU or memory usage, Go's built-in pprof tool is the industry standard.

Recommendation: Expose the pprof endpoints over HTTP.

Implementation:

Import net/http/pprof.

Start a simple HTTP server on a private, internal-only port (e.g., :6060).

Benefits: Allows developers to connect to a running instance of the service to analyze its real-time performance characteristics and generate flame graphs to pinpoint bottlenecks.

4. Advanced Error Handling and Reporting
While the current interceptor logs errors, a more robust system would centralize error reporting.

Recommendation: Integrate an external error tracking service like Sentry or BugSnag.

Implementation:

Modify the logging interceptor to capture and send any non-OK gRPC status codes to the error tracking service.

Include rich context with each error report, such as request metadata and user ID.

Benefits: Provides a centralized dashboard for viewing, triaging, and debugging application errors, complete with stack traces and contextual information.