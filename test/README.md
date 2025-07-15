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

 