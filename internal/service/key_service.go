package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"time"

	"github.com/google/uuid"
	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/kms"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	ErrInvalidRequest    = errors.New("invalid request")
	ErrInvalidKeyType    = errors.New("invalid key type")
	ErrKeyGenerationFail = errors.New("failed to generate cryptographic key")
	ErrMissingMetadata   = errors.New("key metadata is missing")
)

type KeyService interface {
	CreateKey(ctx context.Context, req *pk.CreateKeyRequest) (*pk.CreateKeyResponse, error)
	GetKey(ctx context.Context, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error)
	ListKeys(ctx context.Context, req *pk.ListKeysRequest) (*pk.ListKeysResponse, error)
	RotateKey(ctx context.Context, req *pk.RotateKeyRequest) (*pk.RotateKeyResponse, error)
	RevokeKey(ctx context.Context, req *pk.RevokeKeyRequest) error
	UpdateKeyMetadata(ctx context.Context, req *pk.UpdateKeyMetadataRequest) error
	GetKeyMetadata(ctx context.Context, req *pk.GetKeyMetadataRequest) (*pk.GetKeyMetadataResponse, error)
}

type keyServiceImpl struct {
	keyRepo      domain.KeyRepository
	kmsProviders map[string]kms.KMSProvider
	logger       *slog.Logger
	cfg          *config.Config
}

func NewKeyService(cfg *config.Config, keyRepo domain.KeyRepository, kmsProviders map[string]kms.KMSProvider, logger *slog.Logger) KeyService {
	return &keyServiceImpl{
		cfg:          cfg,
		keyRepo:      keyRepo,
		kmsProviders: kmsProviders,
		logger:       logger,
	}
}

func (s *keyServiceImpl) getKMSProvider(key *domain.Key) (kms.KMSProvider, error) {
	if key == nil {
		return nil, fmt.Errorf("key is nil")
	}
	
	providerName := "local"
	if tier := key.GetTier(); tier == domain.TierPro || tier == domain.TierEnterprise {
		providerName = "aws"
	}
	
	provider, ok := s.kmsProviders[providerName]
	if !ok {
		return nil, fmt.Errorf("%s kms provider not found", providerName)
	}
	return provider, nil
}

func getCryptoDetails(keyType pk.KeyType) (int, string, error) {
	switch keyType {
	case pk.KeyType_KEY_TYPE_AES_256:
		return 32, "AES-256-GCM", nil
	default:
		return 0, "", fmt.Errorf("%w: %s", ErrInvalidKeyType, keyType.String())
	}
}

func validateKeyRequest(keyID string) error {
	if keyID == "" {
		return fmt.Errorf("%w: key ID required", ErrInvalidRequest)
	}
	return nil
}

func (s *keyServiceImpl) getKeyByRequest(ctx context.Context, keyID string, version int32) (*domain.Key, error) {
	if version > 0 {
		return s.keyRepo.GetKeyByVersion(ctx, keyID, version)
	}
	return s.keyRepo.GetKey(ctx, keyID)
}

func (s *keyServiceImpl) GetKey(ctx context.Context, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: request is nil", ErrInvalidRequest)
	}
	if err := validateKeyRequest(req.GetKeyId()); err != nil {
		s.logger.WarnContext(ctx, "invalid get key request")
		return nil, err
	}

	key, err := s.getKeyByRequest(ctx, req.GetKeyId(), req.GetVersion())
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get key", "keyId", req.GetKeyId(), "error", err)
		return nil, fmt.Errorf("failed to get key: %w", err)
	}

	if key.Metadata == nil {
		return nil, ErrMissingMetadata
	}

	resp := &pk.GetKeyResponse{
		KeyMaterial: &pk.KeyMaterial{
			EncryptedKeyData:    append([]byte(nil), key.EncryptedDEK...),
			EncryptionAlgorithm: "AES-256-GCM", 
			KeyChecksum:         "sha256",
		},
		ResponseTimestamp: timestamppb.Now(),
	}

	if !req.GetSkipMetadata() {
		resp.Metadata = key.Metadata
	}

	s.logger.InfoContext(ctx, "key retrieved", "keyId", req.GetKeyId(), "version", key.Version)
	return resp, nil
}

func (s *keyServiceImpl) CreateKey(ctx context.Context, req *pk.CreateKeyRequest) (*pk.CreateKeyResponse, error) {
	if req == nil || req.RequesterContext == nil || req.RequesterContext.GetClientIdentity() == "" {
		s.logger.WarnContext(ctx, "invalid create key request")
		return nil, fmt.Errorf("%w: requester context required", ErrInvalidRequest)
	}

	dekSize, algorithm, err := getCryptoDetails(req.GetKeyType())
	if err != nil {
		s.logger.ErrorContext(ctx, "unsupported key type", "keyType", req.GetKeyType())
		return nil, err
	}

	dek := make([]byte, dekSize)
	if _, err := rand.Read(dek); err != nil {
		s.logger.ErrorContext(ctx, "failed to generate DEK", "error", err)
		return nil, fmt.Errorf("%w: %v", ErrKeyGenerationFail, err)
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
		EncryptedDEK: dek, // Temporary, will be encrypted below
		Status:       domain.KeyStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	kmsProvider, err := s.getKMSProvider(newKey)
	if err != nil {
		return nil, err
	}

	encryptedDEK, err := kmsProvider.EncryptDEK(ctx, newKey)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to encrypt DEK", "error", err)
		return nil, fmt.Errorf("failed to encrypt DEK: %w", err)
	}
	newKey.EncryptedDEK = encryptedDEK

	tier := newKey.GetTier()
	if err := s.keyRepo.CreateKey(ctx, newKey, tier == domain.TierPro || tier == domain.TierEnterprise); err != nil {
		s.logger.ErrorContext(ctx, "failed to create key", "keyId", keyID, "error", err)
		return nil, fmt.Errorf("failed to create key: %w", err)
	}

	resp := &pk.CreateKeyResponse{
		KeyId:    keyID,
		Metadata: metadata,
		KeyMaterial: &pk.KeyMaterial{
			EncryptedKeyData:    append([]byte(nil), encryptedDEK...),
			EncryptionAlgorithm: algorithm,
			KeyChecksum:         "sha256",
		},
		ResponseTimestamp: timestamppb.Now(),
	}

	s.logger.InfoContext(ctx, "key created", "keyId", keyID, "keyType", req.GetKeyType().String(), "creator", req.RequesterContext.GetClientIdentity())
	return resp, nil
}

func (s *keyServiceImpl) ListKeys(ctx context.Context, req *pk.ListKeysRequest) (*pk.ListKeysResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: request is nil", ErrInvalidRequest)
	}

	keys, err := s.keyRepo.ListKeys(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list keys", "error", err)
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	metadataKeys := make([]*pk.KeyMetadata, len(keys))
	for i, key := range keys {
		metadataKeys[i] = key.Metadata
	}

	count := int32(len(metadataKeys))
	resp := &pk.ListKeysResponse{
		Keys:              metadataKeys,
		TotalCount:        count,
		FilteredCount:     count,
		ResponseTimestamp: timestamppb.Now(),
	}

	s.logger.InfoContext(ctx, "keys listed", "count", len(metadataKeys))
	return resp, nil
}

func (s *keyServiceImpl) RotateKey(ctx context.Context, req *pk.RotateKeyRequest) (*pk.RotateKeyResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: request is nil", ErrInvalidRequest)
	}
	if err := validateKeyRequest(req.GetKeyId()); err != nil {
		s.logger.WarnContext(ctx, "invalid rotate key request")
		return nil, err
	}

	currentKey, err := s.keyRepo.GetKey(ctx, req.GetKeyId())
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get current key", "keyId", req.GetKeyId(), "error", err)
		return nil, fmt.Errorf("failed to get current key: %w", err)
	}

	newDEK := make([]byte, 32)
	if _, err := rand.Read(newDEK); err != nil {
		s.logger.ErrorContext(ctx, "failed to generate new DEK", "error", err)
		return nil, fmt.Errorf("%w: %v", ErrKeyGenerationFail, err)
	}

	now := time.Now()
	newKey := &domain.Key{
		ID:           currentKey.ID,
		Version:      currentKey.Version + 1,
		Metadata:     currentKey.Metadata,
		EncryptedDEK: newDEK, // Temporary, will be encrypted below
		Status:       domain.KeyStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	kmsProvider, err := s.getKMSProvider(newKey)
	if err != nil {
		return nil, err
	}

	encryptedNewDEK, err := kmsProvider.EncryptDEK(ctx, newKey)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to encrypt new DEK", "error", err)
		return nil, fmt.Errorf("failed to encrypt new DEK: %w", err)
	}

	rotatedKey, err := s.keyRepo.RotateKey(ctx, req.GetKeyId(), encryptedNewDEK)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to rotate key", "keyId", req.GetKeyId(), "error", err)
		return nil, fmt.Errorf("failed to rotate key: %w", err)
	}

	gracePeriod := time.Duration(req.GetGracePeriodSeconds()) * time.Second
	
	resp := &pk.RotateKeyResponse{
		KeyId:           req.GetKeyId(),
		NewVersion:      rotatedKey.Version,
		PreviousVersion: currentKey.Version,
		NewKeyMaterial: &pk.KeyMaterial{
			EncryptedKeyData:    append([]byte(nil), encryptedNewDEK...),
			EncryptionAlgorithm: "AES-256-GCM",
			KeyChecksum:         "sha256",
		},
		Metadata:            rotatedKey.Metadata,
		RotationTimestamp:   timestamppb.Now(),
		OldVersionExpiresAt: timestamppb.New(now.Add(gracePeriod)),
	}

	s.logger.InfoContext(ctx, "key rotated", "keyId", req.GetKeyId(), "previousVersion", currentKey.Version, "newVersion", rotatedKey.Version)
	return resp, nil
}

func (s *keyServiceImpl) RevokeKey(ctx context.Context, req *pk.RevokeKeyRequest) error {
	if req == nil {
		return fmt.Errorf("%w: request is nil", ErrInvalidRequest)
	}
	if err := validateKeyRequest(req.GetKeyId()); err != nil {
		return err
	}

	if err := s.keyRepo.RevokeKey(ctx, req.GetKeyId()); err != nil {
		s.logger.ErrorContext(ctx, "failed to revoke key", "keyId", req.GetKeyId(), "error", err)
		return fmt.Errorf("failed to revoke key: %w", err)
	}

	s.logger.InfoContext(ctx, "key revoked", "keyId", req.GetKeyId())
	return nil
}

func (s *keyServiceImpl) UpdateKeyMetadata(ctx context.Context, req *pk.UpdateKeyMetadataRequest) error {
	if req == nil {
		return fmt.Errorf("%w: request is nil", ErrInvalidRequest)
	}
	if err := validateKeyRequest(req.GetKeyId()); err != nil {
		return err
	}

	key, err := s.keyRepo.GetKey(ctx, req.GetKeyId())
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get key for metadata update", "keyId", req.GetKeyId(), "error", err)
		return fmt.Errorf("failed to get key: %w", err)
	}

	metadata := key.Metadata
	var updatedFields []string

	if req.Description != nil {
		metadata.Description = *req.Description
		updatedFields = append(updatedFields, "description")
	}
	if req.ExpiresAt != nil {
		metadata.ExpiresAt = req.ExpiresAt
		updatedFields = append(updatedFields, "expiresAt")
	}
	if req.DataClassification != nil {
		metadata.DataClassification = *req.DataClassification
		updatedFields = append(updatedFields, "dataClassification")
	}

	if len(req.GetTagsToAdd()) > 0 || len(req.GetTagsToRemove()) > 0 {
		if metadata.Tags == nil {
			metadata.Tags = make(map[string]string)
		}
		if len(req.GetTagsToAdd()) > 0 {
			maps.Copy(metadata.Tags, req.GetTagsToAdd())
			updatedFields = append(updatedFields, "tags")
		}
		for _, tag := range req.GetTagsToRemove() {
			delete(metadata.Tags, tag)
		}
	}

	metadata.UpdatedAt = timestamppb.Now()

	if err := s.keyRepo.UpdateKeyMetadata(ctx, req.GetKeyId(), metadata); err != nil {
		s.logger.ErrorContext(ctx, "failed to update key metadata", "keyId", req.GetKeyId(), "error", err)
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	s.logger.InfoContext(ctx, "key metadata updated", "keyId", req.GetKeyId(), "fields", updatedFields)
	return nil
}

func (s *keyServiceImpl) GetKeyMetadata(ctx context.Context, req *pk.GetKeyMetadataRequest) (*pk.GetKeyMetadataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: request is nil", ErrInvalidRequest)
	}
	if err := validateKeyRequest(req.GetKeyId()); err != nil {
		return nil, err
	}

	key, err := s.getKeyByRequest(ctx, req.GetKeyId(), req.GetVersion())
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get key metadata", "keyId", req.GetKeyId(), "error", err)
		return nil, fmt.Errorf("failed to get key metadata: %w", err)
	}

	resp := &pk.GetKeyMetadataResponse{
		Metadata:          key.Metadata,
		ResponseTimestamp: timestamppb.Now(),
	}

	if req.GetIncludeAccessHistory() {
		s.logger.WarnContext(ctx, "IncludeAccessHistory not implemented", "keyId", req.GetKeyId())
	}
	if req.GetIncludePolicyDetails() {
		s.logger.WarnContext(ctx, "IncludePolicyDetails not implemented", "keyId", req.GetKeyId())
	}

	s.logger.InfoContext(ctx, "key metadata retrieved", "keyId", req.GetKeyId(), "version", key.Version)
	return resp, nil
}