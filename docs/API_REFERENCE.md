# Polykey API Reference

## 1. Overview

This document provides a detailed technical reference for the `polykey.v2.PolykeyService` gRPC API. It is intended for developers building clients to interact with the Polykey service.

The service exposes endpoints for authentication, health checks, and comprehensive key management operations, including single, batch, and streaming modes.

## 2. Authentication

Clients must first call the `Authenticate` RPC to exchange a pre-configured Client ID and API Key for a JWT Bearer Token. This token must be passed in the `authorization` metadata header for all subsequent API calls.

**Metadata Header Example:**

`authorization: Bearer <jwt-token>`

---

## 3. Service & Authentication RPCs

### HealthCheck

Checks the health of the service.

-   **Request:** `google.protobuf.Empty`
-   **Response:** `HealthCheckResponse`

| Field | Type | Description |
| :--- | :--- | :--- |
| `status` | `HealthStatus` | The health status of the service (e.g., `HEALTHY`, `DEGRADED`). |
| `timestamp` | `google.protobuf.Timestamp` | The timestamp of the health check. |
| `metrics` | `ServiceMetrics` | Key performance metrics of the service. |
| `service_version` | `string` | The version of the running service. |
| `build_commit` | `string` | The git commit hash of the service build. |

### Authenticate

Exchanges a client ID and API key for a JWT access token.

-   **Request:** `AuthenticateRequest`

| Field | Type | Description |
| :--- | :--- | :--- |
| `client_id` | `string` | The client's unique identifier. |
| `api_key` | `string` | The client's pre-shared secret. |

-   **Response:** `AuthenticateResponse`

| Field | Type | Description |
| :--- | :--- | :--- |
| `access_token` | `string` | The JWT access token. |
| `token_type` | `string` | Always "Bearer". |
| `expires_in` | `int64` | The token's time-to-live in seconds. |
| `permissions` | `repeated string` | List of permissions granted to the token. |
| `issued_at` | `google.protobuf.Timestamp` | The time the token was issued. |
| `client_tier` | `common.v2.ClientTier` | The client's service tier. |

---

## 4. Single Key Operation RPCs

### CreateKey

Creates a new cryptographic key.

-   **Request:** `CreateKeyRequest`
-   **Response:** `CreateKeyResponse`

| Field | Type | Description |
| :--- | :--- | :--- |
| `key_id` | `string` | The UUID of the created key. |
| `metadata` | `KeyMetadata` | The metadata of the created key. |
| `key_material` | `KeyMaterial` | The encrypted material of the created key. |
| `response_timestamp` | `google.protobuf.Timestamp` | The timestamp of the response. |

### GetKey

Retrieves a key's material and (optionally) its metadata.

-   **Request:** `GetKeyRequest`
-   **Response:** `GetKeyResponse`

| Field | Type | Description |
| :--- | :--- | :--- |
| `key_material` | `KeyMaterial` | The key's encrypted material. |
| `metadata` | `KeyMetadata` | The key's metadata (omitted if `skip_metadata` was true). |
| `response_timestamp` | `google.protobuf.Timestamp` | The timestamp of the response. |

### GetKeyMetadata

Retrieves the metadata for a specific key.

-   **Request:** `GetKeyMetadataRequest`
-   **Response:** `GetKeyMetadataResponse`

| Field | Type | Description |
| :--- | :--- | :--- |
| `metadata` | `KeyMetadata` | The metadata of the key. |
| `response_timestamp` | `google.protobuf.Timestamp` | The timestamp of the response. |

### ListKeys

Lists keys, returning their metadata with pagination.

-   **Request:** `ListKeysRequest`
-   **Response:** `ListKeysResponse`

| Field | Type | Description |
| :--- | :--- | :--- |
| `keys` | `repeated KeyMetadata` | A list of key metadata objects. |
| `next_page_token` | `string` | A token to retrieve the next page of results. |
| `response_timestamp` | `google.protobuf.Timestamp` | The timestamp of the response. |

### RotateKey

Rotates a key, creating a new version. The old version is kept for a grace period.

-   **Request:** `RotateKeyRequest`
-   **Response:** `RotateKeyResponse`

| Field | Type | Description |
| :--- | :--- | :--- |
| `key_id` | `string` | The UUID of the key. |
| `new_version` | `int32` | The version number of the new key material. |
| `previous_version` | `int32` | The version number of the old key material. |
| `new_key_material` | `KeyMaterial` | The new encrypted key material. |
| `metadata` | `KeyMetadata` | The updated key metadata reflecting the new version. |
| `rotation_timestamp` | `google.protobuf.Timestamp` | The timestamp of the rotation. |
| `old_version_expires_at` | `google.protobuf.Timestamp` | The time when the old key version can no longer be used. |

### RevokeKey

Revokes all versions of a key, making it unusable.

-   **Request:** `RevokeKeyRequest`
-   **Response:** `google.protobuf.Empty`

### UpdateKeyMetadata

Updates the user-definable metadata of a key.

-   **Request:** `UpdateKeyMetadataRequest`
-   **Response:** `google.protobuf.Empty`

---

## 5. Batch Operation RPCs

Batch RPCs allow for processing multiple keys in a single API call. Each result in the response corresponds to a request by its index or key ID and contains either a success message or an error.

-   **`BatchCreateKeys(BatchCreateKeysRequest) returns (BatchCreateKeysResponse)`**
-   **`BatchGetKeys(BatchGetKeysRequest) returns (BatchGetKeysResponse)`**
-   **`BatchGetKeyMetadata(BatchGetKeyMetadataRequest) returns (BatchGetKeyMetadataResponse)`**
-   **`BatchRotateKeys(BatchRotateKeysRequest) returns (BatchRotateKeysResponse)`**
-   **`BatchRevokeKeys(BatchRevokeKeysRequest) returns (BatchRevokeKeysResponse)`**
-   **`BatchUpdateKeyMetadata(BatchUpdateKeyMetadataRequest) returns (BatchUpdateKeyMetadataResponse)`**

---

## 6. Data Models

-   **`KeyMetadata`**: Contains all metadata for a key, including `key_id`, `key_type`, `status`, `version`, timestamps, `creator_identity`, `authorized_contexts`, `tags`, and `storage_type`.
-   **`KeyMaterial`**: Contains the key's cryptographic material, including `encrypted_key_data` and the `encryption_algorithm`.
-   **`RequesterContext`**: Contains information about the client making the request, such as `client_identity`. Used for authorization and auditing.
-   **`AccessAttributes`**: Contains attributes about the access request itself (environment, network zone, etc.) for fine-grained access control.

*(For detailed information on all request and response fields, please refer to the `.proto` definition files.)*