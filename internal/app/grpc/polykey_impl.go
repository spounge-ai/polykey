package grpc

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"

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
		ID:           fmt.Sprintf("key-%d", len(s.keyRepo.ListKeys(ctx))), // Mock ID generation
		Metadata:     req.GetMetadata(),
		EncryptedDEK: encryptedDEK,
	}

	if err := s.keyRepo.CreateKey(ctx, newKey); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create key: %v", err)
	}

	return &pk.CreateKeyResponse{
		KeyId: newKey.ID,
	}, nil
}

// ... other methods ...
