package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/spounge-ai/polykey/internal/config"
	"github.com/spounge-ai/polykey/internal/storage"
	"github.com/spounge-ai/polykey/internal/adapters/security"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// polykeyServiceImpl implements the PolykeyService interface.
type polykeyServiceImpl struct {
	pk.UnimplementedPolykeyServiceServer
	cfg     *config.Config
	storage storage.Storage
}

// NewPolykeyService creates a new instance of PolykeyService.
func NewPolykeyService(cfg *config.Config, s storage.Storage) (pk.PolykeyServiceServer, error) {
	return &polykeyServiceImpl{
		cfg:     cfg,
		storage: s,
	}, nil
}

// GetKey implements pk.PolykeyServiceServer.
func (s *polykeyServiceImpl) GetKey(ctx context.Context, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error) {
	log.Printf("Received GetKey request for key_id: %s", req.GetKeyId())

	// Simulate key retrieval and decryption
	mockKeyMaterial := []byte("mock_key_material_for_" + req.GetKeyId())
	encryptionKey := []byte("thisisatestkeyforpolykeymockserv") // 32 bytes for AES-256

	encryptedKeyData, err := security.Encrypt(encryptionKey, mockKeyMaterial)
	if err != nil {
		log.Printf("Failed to encrypt mock key material: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to process key material")
	}

	return &pk.GetKeyResponse{
		KeyMaterial: &pk.KeyMaterial{
			EncryptedKeyData:    encryptedKeyData,
			EncryptionAlgorithm: "AES-256-GCM",
			KeyDerivationParams: "mock_kdf_params",
			KeyChecksum:         "mock_checksum",
		},
		Metadata: &pk.KeyMetadata{
			KeyId:      req.GetKeyId(),
			KeyType:    pk.KeyType_KEY_TYPE_API_KEY,
			Status:     pk.KeyStatus_KEY_STATUS_ACTIVE,
			Version:    1,
			CreatedAt:  timestamppb.Now(),
			UpdatedAt:  timestamppb.Now(),
			CreatorIdentity: "mock_creator",
		},
		ResponseTimestamp: timestamppb.Now(),
		AuthorizationDecisionId: "mock_auth_decision_id",
	}, nil
}

// ListKeys implements pk.PolykeyServiceServer.
func (s *polykeyServiceImpl) ListKeys(ctx context.Context, req *pk.ListKeysRequest) (*pk.ListKeysResponse, error) {
	log.Println("Received ListKeys request")

	// Simulate listing keys
	var keys []*pk.KeyMetadata
	for i := 0; i < 5; i++ {
		keys = append(keys, &pk.KeyMetadata{
			KeyId:           fmt.Sprintf("mock_key_%d", i),
			KeyType:         pk.KeyType_KEY_TYPE_API_KEY,
			Status:          pk.KeyStatus_KEY_STATUS_ACTIVE,
			Version:         1,
			CreatedAt:       timestamppb.Now(),
			UpdatedAt:       timestamppb.Now(),
			CreatorIdentity: "mock_creator",
		})
	}

	return &pk.ListKeysResponse{
		Keys:              keys,
		NextPageToken:     "",
		TotalCount:        int32(len(keys)),
		ResponseTimestamp: timestamppb.Now(),
		FilteredCount:     int32(len(keys)),
	}, nil
}

// CreateKey implements pk.PolykeyServiceServer.
func (s *polykeyServiceImpl) CreateKey(ctx context.Context, req *pk.CreateKeyRequest) (*pk.CreateKeyResponse, error) {
	log.Printf("Received CreateKey request for key_type: %s", req.GetKeyType().String())

	// Simulate key creation and encryption
	newKeyID := fmt.Sprintf("new_mock_key_%d", time.Now().UnixNano())
	mockKeyMaterial := []byte("new_mock_key_material_for_" + newKeyID)
	encryptionKey := []byte("thisisatestkeyforpolykeymockserv") // 32 bytes for AES-256

	encryptedKeyData, err := security.Encrypt(encryptionKey, mockKeyMaterial)
	if err != nil {
		log.Printf("Failed to encrypt new mock key material: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to process new key material")
	}

	return &pk.CreateKeyResponse{
		KeyId: newKeyID,
		KeyMaterial: &pk.KeyMaterial{
			EncryptedKeyData:    encryptedKeyData,
			EncryptionAlgorithm: "AES-256-GCM",
			KeyDerivationParams: "mock_kdf_params",
			KeyChecksum:         "mock_checksum",
		},
		Metadata: &pk.KeyMetadata{
			KeyId:      newKeyID,
			KeyType:    req.GetKeyType(),
			Status:     pk.KeyStatus_KEY_STATUS_ACTIVE,
			Version:    1,
			CreatedAt:  timestamppb.Now(),
			UpdatedAt:  timestamppb.Now(),
			CreatorIdentity: req.GetRequesterContext().GetClientIdentity(),
			Description: req.GetDescription(),
			Tags: req.GetTags(),
		},
		ResponseTimestamp: timestamppb.Now(),
	}, nil
}

// RotateKey implements pk.PolykeyServiceServer.
func (s *polykeyServiceImpl) RotateKey(ctx context.Context, req *pk.RotateKeyRequest) (*pk.RotateKeyResponse, error) {
	log.Printf("Received RotateKey request for key_id: %s", req.GetKeyId())

	// Simulate key rotation
	newVersion := time.Now().UnixNano() % 100 // Mock new version
	mockNewKeyMaterial := []byte(fmt.Sprintf("rotated_mock_key_material_for_%s_v%d", req.GetKeyId(), newVersion))
	encryptionKey := []byte("thisisatestkeyforpolykeymockserv") // 32 bytes for AES-256

	encryptedNewKeyData, err := security.Encrypt(encryptionKey, mockNewKeyMaterial)
	if err != nil {
		log.Printf("Failed to encrypt new mock key material during rotation: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to process new key material during rotation")
	}

	return &pk.RotateKeyResponse{
		KeyId:             req.GetKeyId(),
		NewVersion:        int32(newVersion),
		PreviousVersion:   0, // Mocking previous version as RotateKeyRequest does not have a version field
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
			CreatedAt:       timestamppb.Now(),
			UpdatedAt:       timestamppb.Now(),
			CreatorIdentity: req.GetRequesterContext().GetClientIdentity(),
		},
		RotationTimestamp:   timestamppb.Now(),
		OldVersionExpiresAt: timestamppb.New(time.Now().Add(time.Hour * 24 * 7)), // Expires in 7 days
	}, nil
}

// RevokeKey implements pk.PolykeyServiceServer.
func (s *polykeyServiceImpl) RevokeKey(ctx context.Context, req *pk.RevokeKeyRequest) (*emptypb.Empty, error) {
	log.Printf("Received RevokeKey request for key_id: %s, reason: %s", req.GetKeyId(), req.GetRevocationReason())
	// Simulate key revocation
	return &emptypb.Empty{}, nil
}

// UpdateKeyMetadata implements pk.PolykeyServiceServer.
func (s *polykeyServiceImpl) UpdateKeyMetadata(ctx context.Context, req *pk.UpdateKeyMetadataRequest) (*emptypb.Empty, error) {
	log.Printf("Received UpdateKeyMetadata request for key_id: %s", req.GetKeyId())
	// Simulate metadata update
	return &emptypb.Empty{}, nil
}

// GetKeyMetadata implements pk.PolykeyServiceServer.
func (s *polykeyServiceImpl) GetKeyMetadata(ctx context.Context, req *pk.GetKeyMetadataRequest) (*pk.GetKeyMetadataResponse, error) {
	log.Printf("Received GetKeyMetadata request for key_id: %s", req.GetKeyId())

	// Simulate metadata retrieval
	return &pk.GetKeyMetadataResponse{
		Metadata: &pk.KeyMetadata{
			KeyId:           req.GetKeyId(),
			KeyType:         pk.KeyType_KEY_TYPE_API_KEY,
			Status:          pk.KeyStatus_KEY_STATUS_ACTIVE,
			Version:         req.GetVersion(),
			CreatedAt:       timestamppb.New(time.Now().Add(-time.Hour * 24 * 30)),
			UpdatedAt:       timestamppb.Now(),
			CreatorIdentity: "mock_creator",
			Description:     "Mock key metadata description",
			Tags:            map[string]string{"env": "mock", "project": "polykey"},
		},
		AccessHistory: []*pk.AccessHistoryEntry{
			{
				Timestamp:    timestamppb.New(time.Now().Add(-time.Hour * 1)),
				ClientIdentity: "mock_client_1",
				Operation:    "GetKey",
				Success:      true,
				CorrelationId: "corr_id_1",
			},
			{
				Timestamp:    timestamppb.New(time.Now().Add(-time.Minute * 30)),
				ClientIdentity: "mock_client_2",
				Operation:    "ListKeys",
				Success:      true,
				CorrelationId: "corr_id_2",
			},
		},
		PolicyDetails: map[string]*pk.PolicyDetail{
			"mock_policy_1": {
				PolicyId:   "policy_id_1",
				PolicyType: "RBAC",
				PolicyParams: map[string]string{"role": "admin"},
			},
		},
		ResponseTimestamp: timestamppb.Now(),
	}, nil
}

// HealthCheck implements pk.PolykeyServiceServer.
func (s *polykeyServiceImpl) HealthCheck(ctx context.Context, req *emptypb.Empty) (*pk.HealthCheckResponse, error) {
	log.Println("Received HealthCheck request")

	// Perform a basic health check on the storage backend
	if s.storage != nil {
		err := s.storage.HealthCheck()
		if err != nil {
			log.Printf("Storage health check failed: %v", err)
			return &pk.HealthCheckResponse{
				Status: pk.HealthStatus_HEALTH_STATUS_UNHEALTHY,
				Timestamp: timestamppb.Now(),
				ServiceVersion: "unknown", // Replace with actual version
				BuildCommit:    "unknown", // Replace with actual commit
			},
			nil
		}
	}

	return &pk.HealthCheckResponse{
		Status: pk.HealthStatus_HEALTH_STATUS_HEALTHY,
		Timestamp: timestamppb.Now(),
		ServiceVersion: "1.0.0", // Example version
		BuildCommit:    "abcdef12345", // Example commit
		Metrics: &pk.ServiceMetrics{
			AverageResponseTimeMs: 10.5,
			RequestsPerSecond:     100,
			ErrorRatePercent:      0.1,
			CpuUsagePercent:       20.0,
			MemoryUsagePercent:    30.0,
			ActiveKeysCount:       1000,
			TotalRequestsHandled:  100000,
			UptimeSince:           timestamppb.New(time.Now().Add(-24 * time.Hour)),
		},
	},
	nil
}
