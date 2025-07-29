package keymanager

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/spounge-ai/polykey/internal/adapters/security"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// KeyManager defines the interface for key management operations.
type KeyManager interface {
	GetKey(ctx context.Context, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error)
	ListKeys(ctx context.Context, req *pk.ListKeysRequest) (*pk.ListKeysResponse, error)
	CreateKey(ctx context.Context, req *pk.CreateKeyRequest) (*pk.CreateKeyResponse, error)
	RotateKey(ctx context.Context, req *pk.RotateKeyRequest) (*pk.RotateKeyResponse, error)
	RevokeKey(ctx context.Context, req *pk.RevokeKeyRequest) error
	UpdateKeyMetadata(ctx context.Context, req *pk.UpdateKeyMetadataRequest) error
	GetKeyMetadata(ctx context.Context, req *pk.GetKeyMetadataRequest) (*pk.GetKeyMetadataResponse, error)
}

// NewKeyManager creates a new instance of KeyManager.
func NewKeyManager() KeyManager {
	return &keyManagerImpl{}
}

// keyManagerImpl implements the KeyManager interface.
type keyManagerImpl struct{}

// GetKey retrieves key material and metadata.
func (km *keyManagerImpl) GetKey(ctx context.Context, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error) {
	// Simulate key retrieval and decryption
	mockKeyMaterial := []byte("mock_key_material_for_" + req.GetKeyId())
	encryptionKey := []byte("thisisatestkeyforpolykeymockserv") // 32 bytes for AES-256

	encryptedKeyData, err := security.Encrypt(encryptionKey, mockKeyMaterial)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt mock key material: %w", err)
	}

	return &pk.GetKeyResponse{
		KeyMaterial: &pk.KeyMaterial{
			EncryptedKeyData:    encryptedKeyData,
			EncryptionAlgorithm: "AES-256-GCM",
			KeyDerivationParams: "mock_kdf_params",
			KeyChecksum:         "mock_checksum",
		},
		Metadata: &pk.KeyMetadata{
			KeyId:           req.GetKeyId(),
			KeyType:         pk.KeyType_KEY_TYPE_API_KEY,
			Status:          pk.KeyStatus_KEY_STATUS_ACTIVE,
			Version:         1,
			CreatedAt:       timestamppb.Now(),
			UpdatedAt:       timestamppb.Now(),
			CreatorIdentity: req.GetRequesterContext().GetClientIdentity(),
		},
		ResponseTimestamp: timestamppb.Now(),
		AuthorizationDecisionId: "mock_auth_decision_id", // This will be set by the service layer
	}, nil
}

// ListKeys enumerates keys accessible to the requesting client.
func (km *keyManagerImpl) ListKeys(ctx context.Context, req *pk.ListKeysRequest) (*pk.ListKeysResponse, error) {
	// Simulate all keys in the system (total_count)
	var allKeys []*pk.KeyMetadata
	for i := 0; i < 10; i++ {
		allKeys = append(allKeys, &pk.KeyMetadata{
			KeyId:           fmt.Sprintf("mock_key_%d", i),
			KeyType:         pk.KeyType_KEY_TYPE_API_KEY,
			Status:          pk.KeyStatus_KEY_STATUS_ACTIVE,
			Version:         1,
			CreatedAt:       timestamppb.Now(),
			UpdatedAt:       timestamppb.Now(),
			CreatorIdentity: "mock_creator",
		})
	}

	// For keymanager, we return all keys, filtering will be done by the service layer
	return &pk.ListKeysResponse{
		Keys:              allKeys,
		NextPageToken:     "",
		TotalCount:        int32(len(allKeys)),
		ResponseTimestamp: timestamppb.Now(),
		FilteredCount:     int32(len(allKeys)), // This will be adjusted by the service layer
	}, nil
}

// CreateKey generates and stores new cryptographic keys.
func (km *keyManagerImpl) CreateKey(ctx context.Context, req *pk.CreateKeyRequest) (*pk.CreateKeyResponse, error) {
	keyType := req.GetKeyType()
	log.Printf("KeyManager: CreateKey - received keyType: %v (%d)", keyType, int(keyType))

	// Simulate secure key generation
	var mockKeyMaterial []byte
	switch keyType {
	case pk.KeyType_KEY_TYPE_API_KEY:
		mockKeyMaterial = fmt.Appendf(nil, "api_key_%d", time.Now().UnixNano())
	case pk.KeyType_KEY_TYPE_AES_256:
		mockKeyMaterial = fmt.Appendf(nil, "aes_key_%d", time.Now().UnixNano())
	case pk.KeyType_KEY_TYPE_RSA_4096:
		mockKeyMaterial = fmt.Appendf(nil, "rsa_key_%d", time.Now().UnixNano())
	case pk.KeyType_KEY_TYPE_ECDSA_P384:
		mockKeyMaterial = fmt.Appendf(nil, "ecdsa_key_%d", time.Now().UnixNano())
	default:
		// This case should ideally not be reached if validation is done at service layer
		return nil, fmt.Errorf("unsupported key type: %v", keyType)
	}

	// Simulate immediate encryption
	encryptionKey := []byte("thisisatestkeyforpolykeymockserv") // 32 bytes for AES-256
	encryptedKeyData, err := security.Encrypt(encryptionKey, mockKeyMaterial)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt new mock key material: %w", err)
	}

	newKeyID := fmt.Sprintf("new_mock_key_%d", time.Now().UnixNano())

	// Establish initial authorization contexts (mocking this for now)
	authorizedContexts := req.GetInitialAuthorizedContexts()
	if len(authorizedContexts) == 0 {
		authorizedContexts = []string{"default_context"}
	}

	resp := &pk.CreateKeyResponse{
		KeyId: newKeyID,
		KeyMaterial: &pk.KeyMaterial{
			EncryptedKeyData:    encryptedKeyData,
			EncryptionAlgorithm: "AES-256-GCM",
			KeyDerivationParams: "mock_kdf_params",
			KeyChecksum:         "mock_checksum",
		},
		Metadata: &pk.KeyMetadata{
			KeyId:               newKeyID,
			KeyType:             keyType,
			Status:              pk.KeyStatus_KEY_STATUS_ACTIVE,
			Version:             1,
			CreatedAt:           timestamppb.Now(),
			UpdatedAt:           timestamppb.Now(),
			CreatorIdentity:     req.GetRequesterContext().GetClientIdentity(),
			Description:         req.GetDescription(),
			Tags:                req.GetTags(),
			AuthorizedContexts:  authorizedContexts,
			AccessPolicies:      req.GetAccessPolicies(),
			DataClassification:  req.GetDataClassification(),
		},
		ResponseTimestamp: timestamppb.Now(),
	}
	log.Printf("KeyManager: CreateKey - returning keyType: %v (%d)", resp.Metadata.KeyType, resp.Metadata.KeyType)
	return resp, nil
}

// RotateKey replaces existing key material.
func (km *keyManagerImpl) RotateKey(ctx context.Context, req *pk.RotateKeyRequest) (*pk.RotateKeyResponse, error) {
	newVersion := time.Now().UnixNano() % 100 // Mock new version
	mockNewKeyMaterial := []byte(fmt.Sprintf("rotated_mock_key_material_for_%s_v%d", req.GetKeyId(), newVersion))
	encryptionKey := []byte("thisisatestkeyforpolykeymockserv") // 32 bytes for AES-256

	encryptedNewKeyData, err := security.Encrypt(encryptionKey, mockNewKeyMaterial)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt new mock key material during rotation: %w", err)
	}

	return &pk.RotateKeyResponse{
		KeyId:             req.GetKeyId(),
		NewVersion:        int32(newVersion),
		PreviousVersion:   1, // Assuming previous version was 1 for mock
		NewKeyMaterial: &pk.KeyMaterial{
			EncryptedKeyData:    encryptedNewKeyData,
			EncryptionAlgorithm: "AES-256-GCM",
			KeyDerivationParams: "mock_kdf_params_rotated",
			KeyChecksum:         "mock_checksum_rotated",
		},
		Metadata: &pk.KeyMetadata{
			KeyId:           req.GetKeyId(),
			KeyType:         pk.KeyType_KEY_TYPE_API_KEY,
			Status:          pk.KeyStatus_KEY_STATUS_ROTATING,
			Version:         int32(newVersion),
			CreatedAt:       timestamppb.New(time.Now().Add(-24 * time.Hour)), // Mock creation time
			UpdatedAt:       timestamppb.Now(),
			CreatorIdentity: req.GetRequesterContext().GetClientIdentity(),
		},
		RotationTimestamp:   timestamppb.Now(),
		OldVersionExpiresAt: timestamppb.New(time.Now().Add(time.Hour * 24 * 7)), // Expires in 7 days
	}, nil
}

// RevokeKey immediately disables key access.
func (km *keyManagerImpl) RevokeKey(ctx context.Context, req *pk.RevokeKeyRequest) error {
	// Simulate key revocation (immediate effect, metadata preservation, secure deletion)
	// In a real system, this would involve updating the key's status in storage
	// and potentially triggering external notifications.
	return nil
}

// UpdateKeyMetadata modifies key metadata and authorization contexts.
func (km *keyManagerImpl) UpdateKeyMetadata(ctx context.Context, req *pk.UpdateKeyMetadataRequest) error {
	// Simulate metadata update
	return nil
}

// GetKeyMetadata retrieves key information without accessing sensitive key material.
func (km *keyManagerImpl) GetKeyMetadata(ctx context.Context, req *pk.GetKeyMetadataRequest) (*pk.GetKeyMetadataResponse, error) {
	resp := &pk.GetKeyMetadataResponse{
		Metadata: &pk.KeyMetadata{
			KeyId:           req.GetKeyId(),
			KeyType:         pk.KeyType_KEY_TYPE_API_KEY,
			Status:          pk.KeyStatus_KEY_STATUS_ACTIVE,
			Version:         req.GetVersion(),
			CreatedAt:       timestamppb.New(time.Now().Add(-time.Hour * 24 * 30)),
			UpdatedAt:       timestamppb.Now(),
			CreatorIdentity: req.GetRequesterContext().GetClientIdentity(),
			Description:     "Mock key metadata description for " + req.GetKeyId(),
			Tags:            map[string]string{"env": "mock", "project": "polykey"},
		},
		ResponseTimestamp: timestamppb.Now(),
	}

	// Optionally include access history
	if req.GetIncludeAccessHistory() {
		resp.AccessHistory = []*pk.AccessHistoryEntry{
			{
				Timestamp:      timestamppb.New(time.Now().Add(-time.Hour * 1)),
				ClientIdentity: "mock_client_1",
				Operation:      "GetKey",
				Success:        true,
				CorrelationId:  "corr_id_1",
			},
			{
				Timestamp:      timestamppb.New(time.Now().Add(-time.Minute * 30)),
				ClientIdentity: "mock_client_2",
				Operation:      "ListKeys",
				Success:        true,
				CorrelationId:  "corr_id_2",
			},
		}
	}

	// Optionally include policy details
	if req.GetIncludePolicyDetails() {
		resp.PolicyDetails = map[string]*pk.PolicyDetail{
			"mock_policy_1": {
				PolicyId:   "policy_id_1",
				PolicyType: "RBAC",
				PolicyParams: map[string]string{"role": "admin"},
			},
		}
	}
	return resp, nil
}
