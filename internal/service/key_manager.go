package service

import (
	"context"
	"fmt"
	"maps"
	"time"

	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/internal/pipelines"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	// Get the current key to determine the storage profile and DEK pool
	currentKey, err := s.keyRepo.GetKey(ctx, keyID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get current key for rotation", "keyId", req.GetKeyId(), "error", err)
		return nil, fmt.Errorf("failed to get current key: %w", err)
	}

	kmsProvider, err := s.getKMSProvider(currentKey.Metadata.GetStorageType())
	if err != nil {
		return nil, err
	}

	dekPool, ok := s.dekPools[currentKey.Metadata.GetKeyType()]
	if !ok {
		return nil, fmt.Errorf("%w: unsupported key type for pooling", ErrInvalidKeyType)
	}

	rotationReq := pipelines.KeyRotationRequest{
		KeyID:       keyID,
		KMSProvider: kmsProvider,
		DEKPool:     dekPool,
	}

	if !s.keyRotationPipeline.Enqueue(rotationReq) {
		return nil, status.Errorf(codes.ResourceExhausted, "key rotation queue is full, please try again later")
	}

	// Wait for the result from the pipeline
	select {
	case result := <-s.keyRotationPipeline.Results():
		if result.Error != nil {
			return nil, result.Error
		}

		rotatedKey := result.RotatedKey
		gracePeriod := time.Duration(req.GetGracePeriodSeconds()) * time.Second
		now := time.Now()

		resp := &pk.RotateKeyResponse{
			KeyId:           req.GetKeyId(),
			NewVersion:      rotatedKey.Version,
			PreviousVersion: currentKey.Version,
			NewKeyMaterial: &pk.KeyMaterial{
				EncryptedKeyData:    append([]byte(nil), rotatedKey.EncryptedDEK...),
				EncryptionAlgorithm: "AES-256-GCM", // This should be dynamic based on key type
				KeyChecksum:         "sha256",
			},
			Metadata:            rotatedKey.Metadata,
			RotationTimestamp:   timestamppb.New(now),
			OldVersionExpiresAt: timestamppb.New(now.Add(gracePeriod)),
		}

		return resp, nil

	case <-ctx.Done():
		return nil, ctx.Err()
	}
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
