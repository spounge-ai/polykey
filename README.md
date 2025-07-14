<div>
  <table>
    <tr>
      <td>
        <img src="SpoungeBordered.png" alt="Polykey Icon" width="100" />
      </td>
      <td style="padding-left: 1em; text-align: left;">
        <div style="font-size: 2.5em; font-weight: bold; line-height: 1.2;">
          Polykey
        </div>
        <div style="font-style: italic; font-size: 1em;">
          A gRPC router microservice for LLMs and third-party APIs, built for the Spounge platform.
        </div>
      </td>
    </tr>
  </table>
</div>

<p>
  <img src="https://img.shields.io/github/actions/workflow/status/SpoungeAI/polykey-service/polykey-build.yml?label=build&style=flat" alt="Build Status" />
  &nbsp;&nbsp;
  <a href="https://github.com/SpoungeAI/polykey-service/pkgs/container/polykey-service" aria-label="Docker image on GHCR">
    <img src="https://img.shields.io/badge/docker%20%7C%20ghcr-blue?logo=docker&style=flat" alt="Docker | GHCR" />
  </a>
  &nbsp;&nbsp;
  <img src="https://img.shields.io/github/go-mod/go-version/SpoungeAI/polykey-service?style=flat" alt="Go Module Version" />
  &nbsp;&nbsp;
  <a href="./LICENSE" aria-label="License MIT">
    <img src="https://img.shields.io/badge/license-MIT-blue.svg?style=flat" alt="License: MIT" />
  </a>
  &nbsp;&nbsp;
  <a href="https://github.com/SpoungeAI/polykey-service/wiki" aria-label="Documentation">
    <img src="https://img.shields.io/badge/docs-polykey--service-blue.svg?style=flat" alt="Docs" />
  </a>
</p>

<hr style="width: 60%; margin: 2em auto 1em;" />






This repository contains:
- The core service logic
- A command-line test client
- A full, containerized development setup using Docker + Make

The project emphasizes **clarity, reproducibility, and extensibility**, with a unified output format for both running and testing the service.

---

## ⚙️ Core Features

- **Modern gRPC:** Robust client-server communication using idiomatic Go and best practices.
- **Containerized Workflow:** Fully containerized via Docker and Docker Compose for consistent dev/test environments.
- **Advanced Makefile:** One Makefile to rule them all—build, test, run, clean, format.
- **Dynamic Config:** Smart config loader supports CLI flags, environment variables, and runtime context.
- **Unified Reporting:** A custom log "beautifier" prints Jest-style output for both app runs and tests.
- **Graceful Shutdown:** Handles system signals (`SIGINT`, `SIGTERM`) for clean exits.

---

## 👨‍💻 Workflow


This project supports two primary development workflows: a fully containerized environment using Docker (recommended) and a local environment using your machine's Go installation.

### Building the Binaries

The `Makefile` provides two distinct build targets:

* **`make build-local` (For Development)**
    This is the **fastest** way to compile the application for local use. It leverages Go's build cache, making subsequent builds nearly instantaneous.
    -   **Use Case:** Quick compilation after making code changes.
    -   **Output:** Executables for your local operating system in the `./bin` directory.

* **`make build` (For Production)**
    This command creates clean, portable, and production-ready binaries. It is slower because it rebuilds all packages from scratch (`-a` flag) and cross-compiles for a `linux/amd64` environment, making it perfect for deploying inside Docker containers.
    -   **Use Case:** Creating the final artifacts for a CI/CD pipeline or for building a Docker image.
    -   **Output:** Statically linked Linux executables in the `./bin` directory.

### Running the Application

#### Docker-Based Workflow (Recommended)

Running the entire stack with Docker Compose is the most reliable way to develop, as it perfectly mirrors a production-like environment.

* **`make compose-dev`**: This single command builds the necessary Docker images and starts both the `polykey-server` and `polykey-dev-client` in isolated containers. Docker Compose handles the networking, ensuring the client can always reach the server.

#### Local Go Workflow

This workflow is useful for quickly testing a single component without the overhead of containerization.

1.  **Start the Server:** In one terminal, run:
    ```sh
    make run-server
    ```
2.  **Run the Client:** In a second terminal, run one of the client commands:
    ```sh
    make run-client
    ```

    This command runs the app and produces a clean summary:
    ```bash

    ▶ RUNS Polykey Dev Client

    ──────────
    SETUP
    ──────────
      ✓ Configuration (server=localhost:50051)

    ──────────
    CONNECTION
    ──────────
      ✓ Network Connectivity
        Connection state changed ... (state=READY)
      ✓ gRPC Connection

    ──────────
    EXECUTION
    ──────────
      ✓ Tool Execution (tool=example_tool)

    ========================================
    PASS  All 4 checks passed
    ```

---

## 🧪 Running Tests

### Available Commands:

| Command                 | Description                                 |
| ----------------------- | ------------------------------------------- |
| `make test`             | Run all unit tests                          |
| `make test-race`        | Run unit tests with the race detector       |
| `make test-integration` | Run integration tests in Docker environment |

Example output:

```
▶ RUNS Go Test Suite

──────────
internal/config
──────────
  ✓ TestDetectRuntime (0.1ms)
  ✓ TestLoadConfig (0.2ms)

cmd/dev_client
──────────
  ✓ TestSomethingInClient (0.1ms)

========================================
 PASS  All 3 tests passed
```

---

## 🧱 Architecture

### 🧩 Configuration

The `ConfigLoader` (`internal/config/config.go`) reads settings from:

* CLI flags (e.g. `-server localhost:50051`)
* Environment variables (e.g. `POLYKEY_SERVER_ADDR`)
* Auto-detection when running inside Docker/K8s

### 🐳 Containerization

* **Dockerfile:** Multi-stage build for lean, secure binaries
* **Docker Compose:** Orchestrates services (`polykey-server`, `polykey-dev-client`) and manages health checks

---

## 🎨 Beautifier Internals

The beautifier is a custom Go formatter (`test/utils/beautify.go`) that parses JSON logs and produces clean, Jest-style reports for both app runs and tests.

It supports logs from the dev client (`make run-client`) and Go tests (`make test`).

For detailed usage and how to extend it, see [test/README.md](test/README.md).

---

## 📄 License

This project is licensed under the MIT License. See [LICENSE](./LICENSE) for details.

```
 