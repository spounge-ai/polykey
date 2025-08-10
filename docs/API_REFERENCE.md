# Polykey API Reference

This document provides a detailed technical reference for the Polykey microservice. It is intended for developers who are building applications that interact with the Polykey API or who are contributing to the Polykey service itself.

## 1. Overview

Polykey is a Key Management Service (KMS) that provides a secure and centralized way to manage cryptographic keys. It exposes a gRPC API for creating, retrieving, rotating, and revoking keys.

## 2. gRPC API

The Polykey gRPC API is defined in protobuf and consists of a single service, `PolykeyService`.

### 2.1 PolykeyService

#### 2.1.1 GetKey

Retrieves a key from the key store.

**Request:** `GetKeyRequest`

| Field | Type | Description |
|---|---|---|
| `key_id` | `string` | The ID of the key to retrieve. |
| `requester_context` | `RequesterContext` | The context of the client making the request. |
| `attributes` | `AccessAttributes` | The attributes of the access request. |
| `version` | `int32` | The version of the key to retrieve. If not specified, the latest version is returned. |
| `skip_metadata` | `bool` | If true, the response will not include the key metadata. |

**Response:** `GetKeyResponse`

| Field | Type | Description |
|---|---|---|
| `key_material` | `KeyMaterial` | The key material. |
| `metadata` | `KeyMetadata` | The key metadata. |
| `response_timestamp` | `google.protobuf.Timestamp` | The timestamp of the response. |
| `authorization_decision_id` | `string` | The ID of the authorization decision. |

#### 2.1.2 ListKeys

Lists keys in the key store.

**Request:** `ListKeysRequest`

| Field | Type | Description |
|---|---|---|
| `requester_context` | `RequesterContext` | The context of the client making the request. |
| `attributes` | `AccessAttributes` | The attributes of the access request. |
| `page_size` | `int32` | The number of keys to return per page. |
| `page_token` | `string` | The page token for pagination. |
| `key_types` | `repeated KeyType` | The types of keys to filter by. |
| `statuses` | `repeated KeyStatus` | The statuses of keys to filter by. |
| `tag_filters` | `map<string, string>` | The tags to filter by. |
| `created_after` | `google.protobuf.Timestamp` | The timestamp to filter keys created after. |
| `created_before` | `google.protobuf.Timestamp` | The timestamp to filter keys created before. |

**Response:** `ListKeysResponse`

| Field | Type | Description |
|---|---|---|
| `keys` | `repeated KeyMetadata` | The list of keys. |
| `next_page_token` | `string` | The next page token for pagination. |
| `total_count` | `int32` | The total number of keys. |
| `response_timestamp` | `google.protobuf.Timestamp` | The timestamp of the response. |
| `filtered_count` | `int32` | The number of keys that matched the filter. |

#### 2.1.3 CreateKey

Creates a new key.

**Request:** `CreateKeyRequest`

| Field | Type | Description |
|---|---|---|
| `key_type` | `KeyType` | The type of key to create. |
| `requester_context` | `RequesterContext` | The context of the client making the request. |
| `description` | `string` | The description of the key. |
| `tags` | `map<string, string>` | The tags to associate with the key. |
| `expires_at` | `google.protobuf.Timestamp` | The expiration timestamp of the key. |
| `initial_authorized_contexts` | `repeated string` | The initial authorized contexts for the key. |
| `access_policies` | `map<string, string>` | The access policies for the key. |
| `data_classification` | `string` | The data classification of the key. |
| `generation_params` | `map<string, string>` | The parameters for key generation. |

**Response:** `CreateKeyResponse`

| Field | Type | Description |
|---|---|---|
| `key_id` | `string` | The ID of the created key. |
| `metadata` | `KeyMetadata` | The metadata of the created key. |
| `key_material` | `KeyMaterial` | The material of the created key. |
| `response_timestamp` | `google.protobuf.Timestamp` | The timestamp of the response. |

#### 2.1.4 RotateKey

Rotates a key.

**Request:** `RotateKeyRequest`

| Field | Type | Description |
|---|---|---|
| `key_id` | `string` | The ID of the key to rotate. |
| `requester_context` | `RequesterContext` | The context of the client making the request. |
| `strategy` | `RotationStrategy` | The rotation strategy. |
| `grace_period_seconds` | `int32` | The grace period in seconds for the old key version. |
| `generation_params` | `map<string, string>` | The parameters for key generation. |
| `preserve_access_policies` | `bool` | If true, the access policies of the old key version will be preserved. |

**Response:** `RotateKeyResponse`

| Field | Type | Description |
|---|---|---|
| `key_id` | `string` | The ID of the rotated key. |
| `new_version` | `int32` | The new version of the key. |
| `previous_version` | `int32` | The previous version of the key. |
| `new_key_material` | `KeyMaterial` | The new key material. |
| `metadata` | `KeyMetadata` | The metadata of the key. |
| `rotation_timestamp` | `google.protobuf.Timestamp` | The timestamp of the rotation. |
| `old_version_expires_at` | `google.protobuf.Timestamp` | The expiration timestamp of the old key version. |

#### 2.1.5 RevokeKey

Revokes a key.

**Request:** `RevokeKeyRequest`

| Field | Type | Description |
|---|---|---|
| `key_id` | `string` | The ID of the key to revoke. |
| `requester_context` | `RequesterContext` | The context of the client making the request. |
| `revocation_reason` | `string` | The reason for revocation. |
| `immediate` | `bool` | If true, the key will be revoked immediately. |
| `effective_at` | `google.protobuf.Timestamp` | The effective timestamp of the revocation. |
| `notify_external_systems` | `bool` | If true, external systems will be notified of the revocation. |

**Response:** `google.protobuf.Empty`

#### 2.1.6 UpdateKeyMetadata

Updates the metadata of a key.

**Request:** `UpdateKeyMetadataRequest`

| Field | Type | Description |
|---|---|---|
| `key_id` | `string` | The ID of the key to update. |
| `requester_context` | `RequesterContext` | The context of the client making the request. |
| `description` | `string` | The new description of the key. |
| `tags_to_add` | `map<string, string>` | The tags to add to the key. |
| `tags_to_remove` | `repeated string` | The tags to remove from the key. |
| `contexts_to_add` | `repeated string` | The contexts to add to the key. |
| `contexts_to_remove` | `repeated string` | The contexts to remove from the key. |
| `policies_to_update` | `map<string, string>` | The policies to update for the key. |
| `expires_at` | `google.protobuf.Timestamp` | The new expiration timestamp of the key. |
| `data_classification` | `string` | The new data classification of the key. |

**Response:** `google.protobuf.Empty`

#### 2.1.7 GetKeyMetadata

Retrieves the metadata of a key.

**Request:** `GetKeyMetadataRequest`

| Field | Type | Description |
|---|---|---|
| `key_id` | `string` | The ID of the key to retrieve metadata for. |
| `requester_context` | `RequesterContext` | The context of the client making the request. |
| `attributes` | `AccessAttributes` | The attributes of the access request. |
| `include_access_history` | `bool` | If true, the response will include the access history of the key. |
| `include_policy_details` | `bool` | If true, the response will include the policy details of the key. |
| `version` | `int32` | The version of the key to retrieve metadata for. If not specified, the latest version is returned. |

**Response:** `GetKeyMetadataResponse`

| Field | Type | Description |
|---|---|---|
| `metadata` | `KeyMetadata` | The metadata of the key. |
| `access_history` | `repeated AccessHistoryEntry` | The access history of the key. |
| `policy_details` | `map<string, PolicyDetail>` | The policy details of the key. |
| `response_timestamp` | `google.protobuf.Timestamp` | The timestamp of the response. |

#### 2.1.8 HealthCheck

Checks the health of the service.

**Request:** `google.protobuf.Empty`

**Response:** `HealthCheckResponse`

| Field | Type | Description |
|---|---|---|
| `status` | `HealthStatus` | The health status of the service. |
| `timestamp` | `google.protobuf.Timestamp` | The timestamp of the health check. |
| `metrics` | `ServiceMetrics` | The metrics of the service. |
| `service_version` | `string` | The version of the service. |
| `build_commit` | `string` | The build commit of the service. |

## 3. Internal Architecture

The Polykey microservice is written in Go and follows a clean architecture pattern.

-   **`cmd`**: Contains the entry points for the server (`polykey`) and a development client (`dev_client`).
-   **`internal`**: Contains the core application logic.
    -   **`app/grpc`**: Implements the gRPC server, including interceptors for authentication and logging.
    -   **`domain`**: Defines the core domain types, such as `Key` and `KeyID`, and interfaces for repositories and services.
    -   **`infra`**: Contains the implementation of infrastructure concerns, such as persistence (NeonDB, S3, CockroachDB), configuration, and authorization.
    -   **`kms`**: Provides an abstraction over Key Management Services, with implementations for AWS KMS and a local KMS.
    -   **`service`**: Implements the business logic of the application, such as key creation, retrieval, and rotation.
    -   **`wiring`**: Handles dependency injection for production and mock environments.
-   **`pkg`**: Contains shared packages that can be used by other applications.

## 4. Configuration

The Polykey service is configured using a YAML file. The following is an example configuration:

```yaml
server:
  port: "<your-server-port>"
  mode: "<your-environment>"
  tls:
    enabled: <true-or-false>
    cert_file: "<path-to-your-cert-file>"
    key_file: "<path-to-your-key-file>"

persistence:
  type: "neondb" # can be "neondb", "s3", or "cockroachdb"

aws:
  enabled: true
  region: "<your-aws-region>"
  s3_bucket: "<your-unique-bucket-name>"
  kms_key_arn: "<your-kms-key-arn>"

vault:
  address: "<your-vault-address>"
  token: "<your-vault-token>"

database:
  host: "<your-database-host>"
  port: "<your-database-port>"
  user: "<your-database-user>"
  password: "<your-database-password>"
  dbname: "polykey"
  sslmode: "require"

neondb:
  url: "postgresql://<user>:<password>@<host>:<port>/<dbname>?sslmode=require"

cockroachdb:
  url: "<your-cockroachdb-connection-string>"

LOCAL_MASTER_KEY: "<your-local-master-key>"

authorization:
  roles:
    admin:
      allowed_operations:
        - "/polykey.v2.PolykeyService/CreateKey"
        - "/polykey.v2.PolykeyService/RotateKey"
        - "/polykey.v2.PolykeyService/RevokeKey"
        - "/polykey.v2.PolykeyService/UpdateKeyMetadata"
        - "/polykey.v2.PolykeyService/GetKey"
        - "/polykey.v2.PolykeyService/GetKeyMetadata"
        - "/polykey.v2.PolykeyService/ListKeys"
    user:
      allowed_operations:
        - "/polykey.v2.PolykeyService/GetKey"
        - "/polykey.v2.PolykeyService/GetKeyMetadata"
        - "/polykey.v2.PolykeyService/ListKeys"
```

## 5. Database Schema

The database schema is defined in SQL migration files.

### `keys` table

```sql
CREATE TABLE IF NOT EXISTS keys (
    id VARCHAR(255) NOT NULL,
    version INT NOT NULL,
    metadata JSONB,
    encrypted_dek BYTEA,
    status VARCHAR(50),
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    is_premium BOOLEAN DEFAULT FALSE,
    PRIMARY KEY (id, version)
);
```

### `audit_events` table

```sql
CREATE TABLE IF NOT EXISTS audit_events (
    id UUID PRIMARY KEY,
    client_identity VARCHAR(255),
    operation VARCHAR(255),
    key_id VARCHAR(255),
    auth_decision_id VARCHAR(255),
    success BOOLEAN,
    error_message TEXT,
    timestamp TIMESTAMPTZ
);
```
