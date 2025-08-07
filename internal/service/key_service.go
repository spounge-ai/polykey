package service

import (
	"context"
	"crypto/rand"
	"log/slog"
	"maps"
	"time"

	"github.com/google/uuid"
	"github.com/spounge-ai/polykey/internal/domain"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// KeyService defines the interface for key management operations.
type KeyService interface {
	CreateKey(ctx context.Context, req *pk.CreateKeyRequest) (*pk.CreateKeyResponse, error)
	GetKey(ctx context.Context, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error)
	ListKeys(ctx context.Context, req *pk.ListKeysRequest) (*pk.ListKeysResponse, error)
	RotateKey(ctx context.Context, req *pk.RotateKeyRequest) (*pk.RotateKeyResponse, error)
	RevokeKey(ctx context.Context, req *pk.RevokeKeyRequest) error
	UpdateKeyMetadata(ctx context.Context, req *pk.UpdateKeyMetadataRequest) error
	GetKeyMetadata(ctx context.Context, req *pk.GetKeyMetadataRequest) (*pk.GetKeyMetadataResponse, error)
}

// keyServiceImpl is the concrete implementation of the KeyService interface.
type keyServiceImpl struct {
	keyRepo domain.KeyRepository
	kms     domain.KMSService
	logger  *slog.Logger
}

// NewKeyService creates a new instance of KeyService.
func NewKeyService(keyRepo domain.KeyRepository, kms domain.KMSService, logger *slog.Logger) KeyService {
	return &keyServiceImpl{
		keyRepo: keyRepo,
		kms:     kms,
		logger:  logger,
	}
}

func (s *keyServiceImpl) GetKey(ctx context.Context, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error) {
	var key *domain.Key
	var err error

	if req.GetVersion() > 0 {
		key, err = s.keyRepo.GetKeyByVersion(ctx, req.GetKeyId(), req.GetVersion())
	} else {
		key, err = s.keyRepo.GetKey(ctx, req.GetKeyId())
	}

	if err != nil {
		return nil, err
	}

	dek, err := s.kms.DecryptDEK(ctx, key.EncryptedDEK, "alias/polykey")
	if err != nil {
		return nil, err
	}

	resp := &pk.GetKeyResponse{
		KeyMaterial: &pk.KeyMaterial{
			EncryptedKeyData:     dek,
			EncryptionAlgorithm:  "AES-256-GCM",
			KeyChecksum:          "sha256",
		},
		Metadata:                 key.Metadata,
		ResponseTimestamp:        timestamppb.Now(),
	}

	if !req.GetSkipMetadata() {
		resp.Metadata = key.Metadata
	}

	return resp, nil
}

func (s *keyServiceImpl) CreateKey(ctx context.Context, req *pk.CreateKeyRequest) (*pk.CreateKeyResponse, error) {
	// Generate DEK
	dek := make([]byte, 32)
	if _, err := rand.Read(dek); err != nil {
		return nil, err
	}

	encryptedDEK, err := s.kms.EncryptDEK(ctx, dek, "alias/polykey")
	if err != nil {
		return nil, err
	}

	keyID := uuid.New().String()
	now := time.Now()

	metadata := &pk.KeyMetadata{
		KeyId:              keyID,
		KeyType:            req.GetKeyType(),
		Status:             pk.KeyStatus_KEY_STATUS_ACTIVE,
		Version:            1,
		CreatedAt:          timestamppb.New(now),
		UpdatedAt:          timestamppb.New(now),
		ExpiresAt:          req.GetExpiresAt(),
		CreatorIdentity:    req.RequesterContext.GetClientIdentity(),
		AuthorizedContexts: req.GetInitialAuthorizedContexts(),
		AccessPolicies:     req.GetAccessPolicies(),
		Description:        req.GetDescription(),
		Tags:               req.GetTags(),
		DataClassification: req.GetDataClassification(),
		AccessCount:        0,
	}

	newKey := &domain.Key{
		ID:           keyID,
		Version:      1,
		Metadata:     metadata,
		EncryptedDEK: encryptedDEK,
		Status:       domain.KeyStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.keyRepo.CreateKey(ctx, newKey); err != nil {
		return nil, err
	}

	resp := &pk.CreateKeyResponse{
		KeyId: keyID,
		Metadata: metadata,
		KeyMaterial: &pk.KeyMaterial{
			EncryptedKeyData:    encryptedDEK,
			EncryptionAlgorithm: "AES-256-GCM",
			KeyChecksum:         "sha256",
		},
		ResponseTimestamp: timestamppb.Now(),
	}

	return resp, nil
}

func (s *keyServiceImpl) ListKeys(ctx context.Context, req *pk.ListKeysRequest) (*pk.ListKeysResponse, error) {
	keys, err := s.keyRepo.ListKeys(ctx)
	if err != nil {
		return nil, err
	}

	var metadataKeys []*pk.KeyMetadata
	for _, key := range keys {
		metadataKeys = append(metadataKeys, key.Metadata)
	}

	resp := &pk.ListKeysResponse{
		Keys:              metadataKeys,
		TotalCount:        int32(len(metadataKeys)),
		FilteredCount:     int32(len(metadataKeys)),
		ResponseTimestamp: timestamppb.Now(),
	}

	return resp, nil
}

func (s *keyServiceImpl) RotateKey(ctx context.Context, req *pk.RotateKeyRequest) (*pk.RotateKeyResponse, error) {
	// Get current key
	currentKey, err := s.keyRepo.GetKey(ctx, req.GetKeyId())
	if err != nil {
		return nil, err
	}

	// Generate new DEK
	newDEK := make([]byte, 32)
	if _, err := rand.Read(newDEK); err != nil {
		return nil, err
	}

	encryptedNewDEK, err := s.kms.EncryptDEK(ctx, newDEK, "alias/polykey")
	if err != nil {
		return nil, err
	}

	// Rotate key in repository
	rotatedKey, err := s.keyRepo.RotateKey(ctx, req.GetKeyId(), encryptedNewDEK)
	if err != nil {
		return nil, err
	}

	resp := &pk.RotateKeyResponse{
		KeyId:           req.GetKeyId(),
		NewVersion:      rotatedKey.Version,
		PreviousVersion: currentKey.Version,
		NewKeyMaterial: &pk.KeyMaterial{
			EncryptedKeyData:    newDEK,
			EncryptionAlgorithm: "AES-256-GCM",
			KeyChecksum:         "sha256",
		},
		Metadata:          rotatedKey.Metadata,
		RotationTimestamp: timestamppb.Now(),
		OldVersionExpiresAt: timestamppb.New(time.Now().Add(time.Duration(req.GetGracePeriodSeconds()) * time.Second)),
	}

	return resp, nil
}

func (s *keyServiceImpl) RevokeKey(ctx context.Context, req *pk.RevokeKeyRequest) error {
	return s.keyRepo.RevokeKey(ctx, req.GetKeyId())
}

func (s *keyServiceImpl) UpdateKeyMetadata(ctx context.Context, req *pk.UpdateKeyMetadataRequest) error {
	// Get current metadata
	key, err := s.keyRepo.GetKey(ctx, req.GetKeyId())
	if err != nil {
		return err
	}

	// Update metadata fields
	metadata := key.Metadata
	if req.Description != nil {
		metadata.Description = *req.Description
	}
	if req.ExpiresAt != nil {
		metadata.ExpiresAt = req.ExpiresAt
	}
	if req.DataClassification != nil {
		metadata.DataClassification = *req.DataClassification
	}

	// Update tags
	if metadata.Tags == nil {
		metadata.Tags = make(map[string]string)
	}
	maps.Copy(metadata.Tags, req.GetTagsToAdd())
	for _, tag := range req.GetTagsToRemove() {
		delete(metadata.Tags, tag)
	}

	metadata.UpdatedAt = timestamppb.Now()

	return s.keyRepo.UpdateKeyMetadata(ctx, req.GetKeyId(), metadata)
}

func (s *keyServiceImpl) GetKeyMetadata(ctx context.Context, req *pk.GetKeyMetadataRequest) (*pk.GetKeyMetadataResponse, error) {
	var key *domain.Key
	var err error

	if req.GetVersion() > 0 {
		key, err = s.keyRepo.GetKeyByVersion(ctx, req.GetKeyId(), req.GetVersion())
	} else {
		key, err = s.keyRepo.GetKey(ctx, req.GetKeyId())
	}

	if err != nil {
		return nil, err
	}

	resp := &pk.GetKeyMetadataResponse{
		Metadata:          key.Metadata,
		ResponseTimestamp: timestamppb.Now(),
	}

	// TODO: Add access history if requested
	if req.GetIncludeAccessHistory() {
		s.logger.Warn("IncludeAccessHistory is not yet implemented")
	}

	// TODO: Add policy details if requested
	if req.GetIncludePolicyDetails() {
		s.logger.Warn("IncludePolicyDetails is not yet implemented")
	}

	return resp, nil
}
