# Polykey Microservice: Function Documentation & Security Architecture

## Overview

This document details the architectural design and security implementation requirements for each function in the Polykey cryptographic key management service. The service implements enterprise-grade security controls aligned with NIST SP 800-57 key management guidance and leverages gRPC with mutual TLS authentication for secure microservice communication.

## Core Security Principles

- **Zero Trust Architecture**: Every request requires explicit authentication and authorization
- **Defense in Depth**: Multiple security layers including mTLS, RBAC/ABAC, audit logging, and encrypted storage
- **Principle of Least Privilege**: Clients receive only keys they are explicitly authorized to access
- **Comprehensive Audit Trail**: All key operations logged with requester identity, timestamps, and context

## Service Functions

### Key Retrieval Operations

#### `GetKey(GetKeyRequest) → GetKeyResponse`

**Purpose**: Primary function for secure key material retrieval with comprehensive authorization.

**Security Implementation**:
- **Authentication**: Validates client certificate through mTLS handshake
- **Authorization**: Enforces RBAC/ABAC policies against `requester_context` and `attributes`
- **Audit Logging**: Records access attempt with client identity, key ID, timestamp, and authorization result
- **Memory Security**: Key material never persisted in logs, temporary variables cleared after response

**Request Parameters**:
- `key_id`: Unique identifier for the requested key
- `requester_context`: Client identity context for authorization evaluation
- `attributes`: Additional ABAC attributes (environment, application, role, etc.)

**Response Structure**:
- `key_material`: Encrypted key data (cleared from server memory immediately after transmission)
- `metadata`: Non-sensitive key information (safe for logging and caching)

**Error Handling**:
- Returns `PERMISSION_DENIED` for unauthorized access attempts
- Returns `NOT_FOUND` for non-existent keys (after authorization check)
- Never exposes key existence in error messages without proper authorization

#### `ListKeys(ListKeysRequest) → ListKeysResponse`

**Purpose**: Enumerate keys accessible to the requesting client with pagination support.

**Security Implementation**:
- **Filtered Response**: Only returns keys the client is authorized to access
- **Metadata Only**: Never includes actual key material in list operations
- **Audit Logging**: Records enumeration requests and filtered result counts

**Pagination Design**:
- Server-controlled page sizes to prevent resource exhaustion
- Opaque continuation tokens to maintain security across paginated requests
- Consistent ordering for reliable pagination behavior

### Key Lifecycle Management

#### `CreateKey(CreateKeyRequest) → CreateKeyResponse`

**Purpose**: Generate and store new cryptographic keys with appropriate metadata and access controls.

**Security Implementation**:
- **Secure Generation**: Uses cryptographically secure random number generation
- **Immediate Encryption**: Key material encrypted before persistence using envelope encryption
- **Access Control Bootstrap**: Establishes initial authorization contexts during creation
- **Audit Trail**: Records key creation with creator identity and initial metadata

**Key Generation Standards**:
- **API Keys**: Cryptographically random 256-bit values, base64-encoded
- **Encryption Keys**: AES-256 keys generated using approved random sources
- **Signing Keys**: RSA-4096 or ECDSA P-384 key pairs with secure private key storage

#### `RotateKey(RotateKeyRequest) → RotateKeyResponse`

**Purpose**: Replace existing key material while maintaining service continuity through versioning.

**Security Implementation**:
- **Version Management**: Maintains multiple key versions during rotation windows
- **Atomic Operations**: Ensures rotation completes fully or rolls back completely
- **Access Preservation**: Maintains existing authorization contexts across rotations
- **Audit Continuity**: Links old and new key versions in audit trail

**Rotation Strategies**:
- **Hot Rotation**: New version becomes primary while old version remains accessible
- **Grace Period**: Configurable overlap period for gradual migration
- **Rollback Capability**: Option to preserve old versions for emergency rollback

#### `RevokeKey(RevokeKeyRequest) → Empty`

**Purpose**: Immediately disable key access while preserving audit trail and metadata.

**Security Implementation**:
- **Immediate Effect**: Key becomes inaccessible to all clients instantly
- **Metadata Preservation**: Maintains key metadata and audit history for compliance
- **Secure Deletion**: Actual key material securely overwritten in storage
- **Cascade Notifications**: Integrates with external systems to invalidate cached keys

#### `UpdateKeyMetadata(UpdateKeyMetadataRequest) → Empty`

**Purpose**: Modify key metadata and authorization contexts without affecting key material.

**Security Implementation**:
- **Authorization Required**: Only authorized administrators can update metadata
- **Change Tracking**: All metadata modifications logged with change details
- **Access Control Updates**: Supports adding/removing authorization contexts
- **Integrity Validation**: Verifies metadata consistency after updates

### Key Information Operations

#### `GetKeyMetadata(GetKeyMetadataRequest) → GetKeyMetadataResponse`

**Purpose**: Retrieve key information without accessing sensitive key material.

**Security Implementation**:
- **Authorization Required**: Still requires proper authorization to view metadata
- **Safe Information**: Response contains only non-sensitive key attributes
- **Audit Logging**: Records metadata access attempts
- **Cache-Friendly**: Response suitable for client-side caching with appropriate TTL

**Metadata Contents**:
- Key type, status, version, and lifecycle timestamps
- Creator identity and custom metadata fields  
- Authorization contexts and expiration information
- Integrity checksums for validation (non-sensitive)

#### `HealthCheck(Empty) → HealthCheckResponse`

**Purpose**: Service health monitoring without exposing sensitive system information.

**Security Implementation**:
- **Minimal Information**: Returns only service availability status
- **No Authentication**: Accessible for monitoring systems without credentials
- **Safe Diagnostics**: Health details exclude sensitive configuration or data
- **Performance Metrics**: Includes response time and basic resource utilization

## Authorization Architecture

### RBAC (Role-Based Access Control)
- **Role Definitions**: Predefined roles mapped to key access patterns
- **Context Evaluation**: Roles derived from client certificate subjects and metadata
- **Hierarchical Permissions**: Role inheritance for administrative capabilities

### ABAC (Attribute-Based Access Control)  
- **Dynamic Evaluation**: Real-time policy decisions based on request attributes
- **Environmental Context**: Time-based, location-based, and network-based restrictions
- **Fine-Grained Control**: Per-key access rules based on multiple attribute combinations

## Audit Logging Requirements

### Mandatory Log Fields
- **Request Timestamp**: High-precision UTC timestamp
- **Client Identity**: X.509 certificate subject and fingerprint  
- **Operation Type**: Specific function called and parameters (excluding key material)
- **Authorization Result**: Success/failure with policy evaluation details
- **Key Identifier**: Target key ID and version
- **Response Metadata**: Success status and error codes (no sensitive data)

### Security Considerations
- **Immutable Logs**: Audit entries cannot be modified after creation
- **Secure Storage**: Audit logs encrypted and integrity-protected
- **Retention Policies**: Long-term retention with secure archival processes
- **No Sensitive Data**: Key material never appears in audit logs

## Error Handling Strategy

### Security-First Error Design
- **Information Disclosure Prevention**: Error messages never reveal unauthorized information
- **Consistent Response Times**: Prevents timing-based information leakage
- **Audit All Failures**: Security-relevant errors always generate audit events
- **Graceful Degradation**: Service remains available during partial failures

### Standard Error Codes
- `UNAUTHENTICATED`: mTLS authentication failure
- `PERMISSION_DENIED`: Authorization policy violation  
- `NOT_FOUND`: Resource not found (after authorization)
- `INVALID_ARGUMENT`: Malformed request parameters
- `INTERNAL`: Service-side errors (details logged but not exposed)

## Implementation Considerations

### Performance Requirements
- **Sub-second Response**: All operations complete within 500ms under normal load
- **Horizontal Scaling**: Stateless design supporting multiple service instances
- **Caching Strategy**: Metadata caching with appropriate invalidation policies
- **Connection Pooling**: Efficient mTLS connection reuse

### Compliance Integration
- **Regulatory Alignment**: Designed for SOX, PCI-DSS, HIPAA compliance requirements
- **Data Residency**: Supports geographic key storage restrictions
- **Key Escrow**: Optional key backup for regulatory compliance
- **Certification Ready**: Architecture supports FIPS 140-2 Level 3 validation

This architectural design prioritizes security, auditability, and operational reliability while maintaining the flexibility required for diverse enterprise key management scenarios.