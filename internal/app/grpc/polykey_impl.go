package grpc

import (
	"context"
	"crypto/rand"

	"github.com/google/uuid"
	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/internal/infra/config"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// polykeyServiceImpl implements the PolykeyService interface.
type polykeyServiceImpl struct {
	pk.UnimplementedPolykeyServiceServer
	cfg        *config.Config
	keyRepo    domain.KeyRepository
	kms        domain.KMSService
	authorizer domain.Authorizer
	audit      domain.AuditLogger
}

// NewPolykeyService creates a new instance of PolykeyService.
func NewPolykeyService(cfg *config.Config, keyRepo domain.KeyRepository, kms domain.KMSService, authorizer domain.Authorizer, audit domain.AuditLogger) (pk.PolykeyServiceServer, error) {
	return &polykeyServiceImpl{
		cfg:        cfg,
		keyRepo:    keyRepo,
		kms:        kms,
		authorizer: authorizer,
		audit:      audit,
	}, nil
}

func (s *polykeyServiceImpl) GetKey(ctx context.Context, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error) {
	key, err := s.keyRepo.GetKey(ctx, req.GetKeyId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve key: %v", err)
	}

	dek, err := s.kms.DecryptDEK(ctx, key.EncryptedDEK, "alias/polykey")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to decrypt key: %v", err)
	}

	resp := &pk.GetKeyResponse{
		KeyMaterial: &pk.KeyMaterial{
			EncryptedKeyData: dek,
		},
		Metadata: key.Metadata,
	}

	return resp, nil
}

func (s *polykeyServiceImpl) CreateKey(ctx context.Context, req *pk.CreateKeyRequest) (*pk.CreateKeyResponse, error) {
	dek := make([]byte, 32)
	if _, err := rand.Read(dek); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate DEK: %v", err)
	}

	encryptedDEK, err := s.kms.EncryptDEK(ctx, dek, "alias/polykey")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to encrypt DEK: %v", err)
	}

	newKey := &domain.Key{
		ID:           uuid.New().String(),
		Metadata:     &pk.KeyMetadata{KeyType: req.GetKeyType()},
		EncryptedDEK: encryptedDEK,
	}

	if err := s.keyRepo.CreateKey(ctx, newKey); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create key: %v", err)
	}

	return &pk.CreateKeyResponse{
		KeyId: newKey.ID,
	}, nil
}

func (s *polykeyServiceImpl) ListKeys(ctx context.Context, req *pk.ListKeysRequest) (*pk.ListKeysResponse, error) {
	keys, err := s.keyRepo.ListKeys(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list keys: %v", err)
	}

	var metadataKeys []*pk.KeyMetadata
	for _, key := range keys {
		metadataKeys = append(metadataKeys, key.Metadata)
	}

	return &pk.ListKeysResponse{
		Keys: metadataKeys,
	}, nil
}

// RotateKey implements pk.PolykeyServiceServer.
func (s *polykeyServiceImpl) RotateKey(ctx context.Context, req *pk.RotateKeyRequest) (*pk.RotateKeyResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RotateKey not implemented")
}

// RevokeKey implements pk.PolykeyServiceServer.
func (s *polykeyServiceImpl) RevokeKey(ctx context.Context, req *pk.RevokeKeyRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RevokeKey not implemented")
}

// UpdateKeyMetadata implements pk.PolykeyServiceServer.
func (s *polykeyServiceImpl) UpdateKeyMetadata(ctx context.Context, req *pk.UpdateKeyMetadataRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateKeyMetadata not implemented")
}

// GetKeyMetadata implements pk.PolykeyServiceServer.
func (s *polykeyServiceImpl) GetKeyMetadata(ctx context.Context, req *pk.GetKeyMetadataRequest) (*pk.GetKeyMetadataResponse, error) {
	key, err := s.keyRepo.GetKey(ctx, req.GetKeyId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve key metadata: %v", err)
	}

	resp := &pk.GetKeyMetadataResponse{
		Metadata: key.Metadata,
	}

	if req.GetIncludeAccessHistory() {
		// TODO: Implement audit log retrieval
	}

	if req.GetIncludePolicyDetails() {
		// TODO: Implement policy retrieval
	}

	return resp, nil
}

// HealthCheck implements pk.PolykeyServiceServer.
func (s *polykeyServiceImpl) HealthCheck(ctx context.Context, req *emptypb.Empty) (*pk.HealthCheckResponse, error) {
	return &pk.HealthCheckResponse{
		Status: pk.HealthStatus_HEALTH_STATUS_HEALTHY,
	}, nil
}