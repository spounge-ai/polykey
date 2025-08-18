
<div align="center">
<picture>
  <source media="(prefers-color-scheme: light)" srcset="./docs/dark.svg">
  <img alt="spounge-polykey-logo" src="./docs/light.svg" width="50%" height="50%">
</picture>

  A gRPC router microservice for LLMs and third-party APIs for [@Spounge](https://github.com/spounge-ai).

[![Build Status](https://img.shields.io/github/actions/workflow/status/SpoungeAI/polykey-service/ci.yml?label=Build&style=flat&color=brightgreen)](https://github.com/SpoungeAI/polykey-service/actions/workflows/ci.yml)
[![Docker | GHCR](https://img.shields.io/badge/Docker-GHCR-blue?logo=docker&style=flat)](https://github.com/SpoungeAI/polykey-service/pkgs/container/polykey-service)
[![Go Module Version](https://img.shields.io/github/go-mod/go-version/SpoungeAI/polykey-service?style=flat)](https://github.com/SpoungeAI/polykey-service/blob/main/go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue?style=flat)](./LICENSE)
[![Docs](https://img.shields.io/badge/docs-polykey--service-blue?style=flat)](https://github.com/SpoungeAI/polykey-service/wiki)


</div>


> Note (08/18/25): Finishing up unit and integration testing then containerizing.

<hr style="width: 100%; height: 4px; background-color: #888; border: none; margin: 2em auto 1em;" />

Polykey is a high-performance, secure-by-design Key Management Service (KMS) written in Go. It provides a gRPC API for the complete lifecycle management of cryptographic keys, built on a cloud-native and observable architecture.

For a detailed technical breakdown of all RPCs, see the [**API Reference**](./docs/API_REFERENCE.md).

For instructions on how to configure the service and build a client, see the [**Integration Guide**](./docs/INTEGRATION_GUIDE.md).

## Core Features

-   **Secure by Design**:
    -   **Transport Security**: Enforces mutual TLS (mTLS) for all gRPC communication.
    -   **Authentication**: Uses a JWT-based authentication flow (`Authenticate` RPC).
    -   **Authorization**: Implements Role-Based Access Control (RBAC) and can enforce a zero-trust policy by matching a client's mTLS certificate identity to its application-layer identity.
    -   **Secure Bootstrapping**: Loads critical secrets (e.g., master keys, JWT signing keys) at startup from **AWS Parameter Store**, not from config files.

-   **Comprehensive Key Lifecycle Management**:
    -   **Full CRUD Operations**: Create, Get, List, Update, and Revoke keys.
    -   **Key Rotation**: Built-in RPC for rotating keys to create new versions.
    -   **Batch Operations**: High-throughput batch endpoints for all major key operations to reduce network latency.

-   **Cloud-Native & Resilient Architecture**:
    -   **Structured Logging**: Uses `slog` for structured, context-aware JSON logging.
    -   **Observability**: Integrated OpenTelemetry for distributed tracing.
    -   **Resilience Patterns**: Employs a circuit breaker pattern for the persistence layer to prevent cascading failures.
    -   **Concurrency**: Uses worker pools and pipelines for handling asynchronous background tasks like key rotation.

-   **Flexible Backend Support**:
    -   **Persistence**: Supports PostgreSQL-compatible databases (like NeonDB and CockroachDB).
    -   **KMS Providers**: Supports a local, master-key-based KMS and AWS Key Management Service (KMS) for hardened, production-grade key encryption.

## Architecture Overview

The service follows a clean architecture pattern, separating concerns into distinct layers within the `internal` directory:

-   `/domain`: Defines the core business entities, interfaces, and value objects.
-   `/service`: Implements the core business logic for key management operations.
-   `/infra`: Contains implementations for external concerns like persistence, caching, and communication with AWS.
-   `/app`: Holds the gRPC server implementation, including all RPC handlers and interceptors for concerns like logging, auth, and validation.
-   `/wiring`: Manages dependency injection for production and mock environments.

## ⚙️ Getting Started

### Building the Binaries

-   `make build-local`: The fastest way to compile for local development, using the Go build cache.
-   `make build`: Creates clean, statically-linked Linux binaries suitable for production/container environments.

### Running the Application

The recommended workflow is using Docker Compose, which mirrors a production-like environment.

-   `make compose-dev`: Builds the necessary Docker images and starts the `polykey-server`.

### Running Tests

| Command | Description |
| :--- | :--- |
| `make test` | Run all unit tests. |
| `make test-race` | Run unit tests with the race detector enabled. |
| `make test-integration` | Run the integration test suite. |

## 📄 License

This project is licensed under the MIT License. See [LICENSE](./LICENSE) for details.

