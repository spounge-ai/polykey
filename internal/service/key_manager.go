package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"maps"
	"time"

	"github.com/spounge-ai/polykey/internal/domain"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *keyServiceImpl) RotateKey(ctx context.Context, req *pk.RotateKeyRequest) (*pk.RotateKeyResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: request is nil", ErrInvalidRequest)
	}
	keyID, err := domain.KeyIDFromString(req.GetKeyId())
	if err != nil {
		return nil, err
	}

	currentKey, err := s.keyRepo.GetKey(ctx, keyID)
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

	encryptedNewDEK, err := kmsProvider.EncryptDEK(ctx, newDEK, newKey)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to encrypt new DEK", "error", err)
		return nil, fmt.Errorf("failed to encrypt new DEK: %w", err)
	}
	defer zeroBytes(newDEK)

	rotatedKey, err := s.keyRepo.RotateKey(ctx, keyID, encryptedNewDEK)
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
	keyID, err := domain.KeyIDFromString(req.GetKeyId())
	if err != nil {
		return err
	}

	if err := s.keyRepo.RevokeKey(ctx, keyID); err != nil {
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
	keyID, err := domain.KeyIDFromString(req.GetKeyId())
	if err != nil {
		return err
	}

	key, err := s.keyRepo.GetKey(ctx, keyID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get key for metadata update", "keyId", req.GetKeyId(), "error", err)
		return fmt.Errorf("failed to get key: %w", err)
	}

	metadata := key.Metadata
	var updatedFields []string

	if req.Description != nil {
		description, err := domain.NewDescription(*req.Description)
		if err != nil {
			return err
		}
		metadata.Description = description.String()
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

	if err := s.keyRepo.UpdateKeyMetadata(ctx, keyID, metadata); err != nil {
		s.logger.ErrorContext(ctx, "failed to update key metadata", "keyId", req.GetKeyId(), "error", err)
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	s.logger.InfoContext(ctx, "key metadata updated", "keyId", req.GetKeyId(), "fields", updatedFields)
	return nil
}
