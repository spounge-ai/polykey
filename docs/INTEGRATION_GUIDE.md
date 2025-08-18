# Polykey Integration Guide

This guide explains how to configure the Polykey service, its security model, and how to build a client microservice to securely connect and interact with it.

## 1. Security Model & Bootstrapping

Polykey is designed to be bootstrapped securely, ensuring that sensitive parameters like master keys and JWT signing keys are never stored in version-controlled configuration files.

### Secret Management

At startup, the service loads its base configuration from `config.yaml`. If configured for an AWS environment, it then fetches a set of critical bootstrap secrets from **AWS Systems Manager Parameter Store**. This process is handled by the component at `internal/infra/secrets/parameter_store.go`.

These bootstrap secrets include:

-   `/polykey/polykey_master_key`: The master key used by the `local` KMS provider for encrypting all Data Encryption Keys (DEKs).
-   `/polykey/jwt_secret`: The RSA private key used to sign all JWTs issued by the `Authenticate` endpoint.
-   `/polykey/databases/neondb_url_development`: The database connection string.

This ensures that the most sensitive secrets are managed securely outside the application's codebase and configuration files.

### Authentication & Authorization Flow

1.  **Transport Layer (mTLS)**: All communication between a client and the Polykey server is secured using mutual TLS. The server verifies the client's certificate against a trusted Certificate Authority (CA). For enhanced security, the server can be configured to match the client certificate's Common Name (CN) with the `client_id` presented during authentication.

2.  **Application Layer (JWT)**: After establishing a secure channel, the client must call the `Authenticate` RPC with its `client_id` and a pre-shared `api_key`. Upon success, the server issues a short-lived JWT Bearer Token. This token must be included in the `authorization` header of all subsequent API calls.

## 2. Service Configuration

Polykey is configured using a YAML file (e.g., `configs/config.example.yaml`).

### Key Configuration Sections

-   **`server.tls`**: To enable mTLS, set `enabled: true` and provide paths to the server certificate, key, and the CA certificate used to validate client certs.
-   **`aws.enabled`**: Must be `true` to enable bootstrapping from AWS Parameter Store and to use the AWS KMS provider.
-   **`client_credentials_path`**: **(Security Critical)** The path to the YAML file containing client identities and their bcrypt-hashed API keys. This is how you register clients that can authenticate with the service.
-   **`authorization.zero_trust.enforce_mtls_identity_match`**: Set to `true` to enforce that the client certificate's Common Name matches the authenticated client ID.

## 3. Building a Client

This section provides a language-agnostic guide to building a client microservice.

### Step 1: Generate Client Credentials

1.  **Generate a strong API key** for your client.
2.  **Hash the API key** using a standard bcrypt library.
3.  **Register the client** by adding an entry to the file specified by `client_credentials_path` in the Polykey server's configuration. For example, for a client named `billing-service`:

    ```yaml
    # In config.client.dev.yaml or similar
    clients:
      billing-service:
        hashed_api_key: "<your-bcrypt-hash>"
        permissions: ["keys:create", "keys:read"]
        tier: "pro"
    ```

### Step 2: Generate Protobuf Client

Use the Polykey `.proto` files and `protoc` to generate a gRPC client for your target language (e.g., Go, Python, TypeScript).

### Step 3: Implement the Connection Flow

1.  **Load TLS Assets**: In your client application, load three TLS files:
    -   The client's public certificate (`client-cert.pem`).
    -   The client's private key (`client-key.pem`).
    -   The Certificate Authority (CA) certificate that signed the Polykey server's certificate (`server-ca.pem`). This is used to verify the server's identity.

2.  **Establish Secure gRPC Connection**: Use your language's gRPC library to create a secure, mTLS-encrypted connection to the Polykey server, using the TLS assets loaded in the previous step.

3.  **Authenticate and Manage Tokens**: Implement a function to perform the following logic:
    -   Call the `Authenticate` RPC with your client's ID and secret API key.
    -   Receive the `AuthenticateResponse` and store the `access_token` in memory.
    -   The token has a limited lifetime (`expires_in`). Your client should be prepared to re-authenticate when the token expires or before making a request if the token is nearing expiration.

4.  **Make Authorized API Calls**: For all other API calls (e.g., `CreateKey`, `GetKey`):
    -   Create a gRPC metadata/header object.
    -   Add the JWT to the metadata with the key `authorization` and the value `Bearer <your-jwt>`.
    -   Attach the metadata to your outgoing RPC request.