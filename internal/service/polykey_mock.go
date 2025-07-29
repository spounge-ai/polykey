package service

import (
	"context"
	"fmt"
	"log"
	"time"

	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MockPolykeyService implements the PolykeyService interface for testing purposes.
type MockPolykeyService struct{
	pk.UnimplementedPolykeyServiceServer
}

// GetKey is a mock implementation.
func (s *MockPolykeyService) GetKey(ctx context.Context, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error) {
	log.Printf("MockPolykeyService: GetKey called for key_id: %s", req.GetKeyId())
	return &pk.GetKeyResponse{
		KeyMaterial: &pk.KeyMaterial{
			EncryptedKeyData:    []byte("mock_encrypted_key_data"),
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

// ListKeys is a mock implementation.
func (s *MockPolykeyService) ListKeys(ctx context.Context, req *pk.ListKeysRequest) (*pk.ListKeysResponse, error) {
	log.Println("MockPolykeyService: ListKeys called")
	var keys []*pk.KeyMetadata
	for i := 0; i < 3; i++ {
		keys = append(keys, &pk.KeyMetadata{
			KeyId:           fmt.Sprintf("mock_key_%d", i),
			KeyType:         pk.KeyType_KEY_TYPE_AES_256,
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

// CreateKey is a mock implementation.
func (s *MockPolykeyService) CreateKey(ctx context.Context, req *pk.CreateKeyRequest) (*pk.CreateKeyResponse, error) {
	log.Printf("MockPolykeyService: CreateKey called for key_type: %s", req.GetKeyType().String())
	newKeyID := fmt.Sprintf("new_mock_key_%d", time.Now().UnixNano())
	return &pk.CreateKeyResponse{
		KeyId: newKeyID,
		KeyMaterial: &pk.KeyMaterial{
			EncryptedKeyData:    []byte("new_mock_encrypted_key_data"),
			EncryptionAlgorithm: "AES-256-GCM",
			KeyDerivationParams: "mock_kdf_params",
			KeyChecksum:         "mock_checksum",
		},
		Metadata: &pk.KeyMetadata{
			KeyId:           newKeyID,
			KeyType:         req.GetKeyType(),
			Status:          pk.KeyStatus_KEY_STATUS_ACTIVE,
			Version:         1,
			CreatedAt:       timestamppb.Now(),
			UpdatedAt:       timestamppb.Now(),
			CreatorIdentity: req.GetRequesterContext().GetClientIdentity(),
			Description:     req.GetDescription(),
			Tags:            req.GetTags(),
		},
		ResponseTimestamp: timestamppb.Now(),
	}, nil
}

// RotateKey is a mock implementation.
func (s *MockPolykeyService) RotateKey(ctx context.Context, req *pk.RotateKeyRequest) (*pk.RotateKeyResponse, error) {
	log.Printf("MockPolykeyService: RotateKey called for key_id: %s", req.GetKeyId())
	newVersion := time.Now().UnixNano() % 100
	return &pk.RotateKeyResponse{
		KeyId:             req.GetKeyId(),
		NewVersion:        int32(newVersion),
		PreviousVersion:   0,
		NewKeyMaterial: &pk.KeyMaterial{
			EncryptedKeyData:    []byte("rotated_mock_encrypted_key_data"),
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
		OldVersionExpiresAt: timestamppb.New(time.Now().Add(time.Hour * 24 * 7)),
	}, nil
}

// RevokeKey is a mock implementation.
func (s *MockPolykeyService) RevokeKey(ctx context.Context, req *pk.RevokeKeyRequest) (*emptypb.Empty, error) {
	log.Printf("MockPolykeyService: RevokeKey called for key_id: %s, reason: %s", req.GetKeyId(), req.GetRevocationReason())
	return &emptypb.Empty{}, nil
}

// UpdateKeyMetadata is a mock implementation.
func (s *MockPolykeyService) UpdateKeyMetadata(ctx context.Context, req *pk.UpdateKeyMetadataRequest) (*emptypb.Empty, error) {
	log.Printf("MockPolykeyService: UpdateKeyMetadata called for key_id: %s", req.GetKeyId())
	return &emptypb.Empty{}, nil
}

// GetKeyMetadata is a mock implementation.
func (s *MockPolykeyService) GetKeyMetadata(ctx context.Context, req *pk.GetKeyMetadataRequest) (*pk.GetKeyMetadataResponse, error) {
	log.Printf("MockPolykeyService: GetKeyMetadata called for key_id: %s", req.GetKeyId())
	return &pk.GetKeyMetadataResponse{
		Metadata: &pk.KeyMetadata{
			KeyId:           req.GetKeyId(),
			KeyType:         pk.KeyType_KEY_TYPE_API_KEY,
			Status:          pk.KeyStatus_KEY_STATUS_ACTIVE,
			Version:         0,
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

// HealthCheck provides a mock health check response.
func (s *MockPolykeyService) HealthCheck(ctx context.Context, req *emptypb.Empty) (*pk.HealthCheckResponse, error) {
	log.Printf("MockPolykeyService: HealthCheck called!")
	return &pk.HealthCheckResponse{
		Status:    pk.HealthStatus_HEALTH_STATUS_HEALTHY,
		Timestamp: timestamppb.New(time.Now()),
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
		ServiceVersion: "v0.0.1-mock",
		BuildCommit:    "mock-commit-12345",
	}, nil
}

