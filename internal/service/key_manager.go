package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"maps"
	"time"

	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/internal/pipelines"
	"github.com/spounge-ai/polykey/pkg/patterns/batch"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// processRotation contains the core logic for rotating a single key.
// It is designed to be called by both single and batch rotation methods.
func (s *keyServiceImpl) processRotation(ctx context.Context, keyID domain.KeyID) (*domain.Key, *domain.Key, error) {
	currentKey, err := s.keyRepo.GetKey(ctx, keyID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get current key for rotation", "keyId", keyID, "error", err)
		return nil, nil, fmt.Errorf("failed to get current key: %w", err)
	}

	kmsProvider, err := s.getKMSProvider(currentKey.Metadata.GetStorageType())
	if err != nil {
		return nil, nil, err
	}

	dekPool, ok := s.dekPools[currentKey.Metadata.GetKeyType()]
	if !ok {
		return nil, nil, fmt.Errorf("%w: unsupported key type for pooling", ErrInvalidKeyType)
	}

	newDEK := dekPool.Get()
	defer dekPool.Put(newDEK)

	if _, err := rand.Read(newDEK); err != nil {
		s.logger.ErrorContext(ctx, "failed to generate new DEK", "error", err)
		return nil, nil, fmt.Errorf("failed to generate new DEK: %w", err)
	}

	encryptedNewDEK, err := kmsProvider.EncryptDEK(ctx, newDEK, currentKey)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to encrypt new DEK", "error", err)
		return nil, nil, fmt.Errorf("failed to encrypt new DEK: %w", err)
	}

	rotatedKey, err := s.keyRepo.RotateKey(ctx, keyID, encryptedNewDEK)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to rotate key in repository", "keyId", keyID, "error", err)
		return nil, nil, fmt.Errorf("failed to rotate key: %w", err)
	}

	s.logger.InfoContext(ctx, "key rotated successfully", "keyId", keyID, "newVersion", rotatedKey.Version)
	return currentKey, rotatedKey, nil
}

func (s *keyServiceImpl) RotateKey(ctx context.Context, req *pk.RotateKeyRequest) (*pk.RotateKeyResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: request is nil", ErrInvalidRequest)
	}
	keyID, err := domain.KeyIDFromString(req.GetKeyId())
	if err != nil {
		return nil, err
	}

	// The pipeline is suitable for single, async-style requests.
	// For a simple RPC, we can also call the logic directly.
	// Here we demonstrate using the pipeline.
	currentKey, err := s.keyRepo.GetKey(ctx, keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current key for rotation: %w", err)
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

func (s *keyServiceImpl) BatchRotateKeys(ctx context.Context, req *pk.BatchRotateKeysRequest) (*pk.BatchRotateKeysResponse, error) {
	ctx, span := tracer.Start(ctx, "BatchRotateKeys")
	defer span.End()

	if req == nil || req.RequesterContext == nil || req.RequesterContext.GetClientIdentity() == "" {
		return nil, ErrInvalidRequest
	}

	processor := batch.BatchProcessor[*pk.RotateKeyItem, *pk.RotateKeyResponse]{
		MaxConcurrency: 10, // Make this configurable
		Validate: func(item *pk.RotateKeyItem) error {
			_, err := domain.KeyIDFromString(item.GetKeyId())
			return err
		},
		Process: func(ctx context.Context, item *pk.RotateKeyItem) (*pk.RotateKeyResponse, error) {
			keyID, _ := domain.KeyIDFromString(item.GetKeyId())
			currentKey, rotatedKey, err := s.processRotation(ctx, keyID)
			if err != nil {
				return nil, err
			}

			gracePeriod := time.Duration(item.GetGracePeriodSeconds()) * time.Second
			now := time.Now()

			return &pk.RotateKeyResponse{
				KeyId:           item.GetKeyId(),
				NewVersion:      rotatedKey.Version,
				PreviousVersion: currentKey.Version,
				NewKeyMaterial: &pk.KeyMaterial{
					EncryptedKeyData:    append([]byte(nil), rotatedKey.EncryptedDEK...),
					EncryptionAlgorithm: "AES-256-GCM", // This should be dynamic
					KeyChecksum:         "sha256",
				},
				Metadata:            rotatedKey.Metadata,
				RotationTimestamp:   timestamppb.New(now),
				OldVersionExpiresAt: timestamppb.New(now.Add(gracePeriod)),
			}, nil
		},
	}

	batchResult, err := processor.ProcessBatch(ctx, req.Keys, req.GetContinueOnError())
	if err != nil {
		return nil, err
	}

	results := make([]*pk.BatchRotateKeysResult, len(batchResult.Items))
	var successCount, failedCount int32
	for i, item := range batchResult.Items {
		if item.Error != nil {
			failedCount++
			results[i] = &pk.BatchRotateKeysResult{
				KeyId:  req.Keys[i].GetKeyId(),
				Result: &pk.BatchRotateKeysResult_Error{Error: item.Error.Error()},
			}
		} else {
			successCount++
			results[i] = &pk.BatchRotateKeysResult{
				KeyId:  req.Keys[i].GetKeyId(),
				Result: &pk.BatchRotateKeysResult_Success{Success: item.Result},
			}
		}
	}

	return &pk.BatchRotateKeysResponse{
		Results:           results,
		ResponseTimestamp: timestamppb.Now(),
		SuccessfulCount:   successCount,
		FailedCount:       failedCount,
	}, nil
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

func (s *keyServiceImpl) BatchRevokeKeys(ctx context.Context, req *pk.BatchRevokeKeysRequest) (*pk.BatchRevokeKeysResponse, error) {
	ctx, span := tracer.Start(ctx, "BatchRevokeKeys")
	defer span.End()

	if req == nil || req.RequesterContext == nil || req.RequesterContext.GetClientIdentity() == "" {
		return nil, ErrInvalidRequest
	}

	var (
		successCount int32
		failedCount  int32
		results      []*pk.BatchRevokeKeysResult
	)

	keyIDs := make([]domain.KeyID, 0, len(req.GetKeys()))
	keyIDMap := make(map[string]bool)
	for _, item := range req.GetKeys() {
		keyID, err := domain.KeyIDFromString(item.GetKeyId())
		if err != nil {
			failedCount++
			results = append(results, &pk.BatchRevokeKeysResult{
				KeyId:  item.GetKeyId(),
				Result: &pk.BatchRevokeKeysResult_Error{Error: err.Error()},
			})
			if !req.GetContinueOnError() {
				return nil, fmt.Errorf("invalid key ID in batch request: %w", err)
			}
			continue
		}
		if !keyIDMap[item.GetKeyId()] {
			keyIDs = append(keyIDs, keyID)
			keyIDMap[item.GetKeyId()] = true
		}
	}

	if err := s.keyRepo.RevokeBatchKeys(ctx, keyIDs); err != nil {
		// If the entire batch revoke fails, return a single error unless continue_on_error is true
		if !req.GetContinueOnError() {
			return nil, fmt.Errorf("failed to revoke batch keys from repository: %w", err)
		}
		// If continue_on_error is true, mark all as failed
		for _, id := range keyIDs {
			failedCount++
			results = append(results, &pk.BatchRevokeKeysResult{
				KeyId:  id.String(),
				Result: &pk.BatchRevokeKeysResult_Error{Error: err.Error()},
			})
		}
		return &pk.BatchRevokeKeysResponse{
			Results:           results,
			ResponseTimestamp: timestamppb.Now(),
			SuccessfulCount:   successCount,
			FailedCount:       failedCount,
		}, nil
	}

	// For successful batch revoke, we assume all keys were revoked.
	// The repository doesn't return individual success/failure for batch.
	for _, id := range keyIDs {
		successCount++
		results = append(results, &pk.BatchRevokeKeysResult{
			KeyId:  id.String(),
			Result: &pk.BatchRevokeKeysResult_Success{Success: true},
		})
	}

	return &pk.BatchRevokeKeysResponse{
		Results:           results,
		ResponseTimestamp: timestamppb.Now(),
		SuccessfulCount:   successCount,
		FailedCount:       failedCount,
	}, nil
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

func (s *keyServiceImpl) BatchUpdateKeyMetadata(ctx context.Context, req *pk.BatchUpdateKeyMetadataRequest) (*pk.BatchUpdateKeyMetadataResponse, error) {
	ctx, span := tracer.Start(ctx, "BatchUpdateKeyMetadata")
	defer span.End()

	if req == nil || req.RequesterContext == nil || req.RequesterContext.GetClientIdentity() == "" {
		return nil, ErrInvalidRequest
	}

	var (
		successCount int32
		failedCount  int32
		results      []*pk.BatchUpdateKeyMetadataResult
	)

	keysToUpdate := make([]*domain.Key, 0, len(req.GetKeys()))
	for _, item := range req.GetKeys() {
		keyID, err := domain.KeyIDFromString(item.GetKeyId())
		if err != nil {
			failedCount++
			results = append(results, &pk.BatchUpdateKeyMetadataResult{
				KeyId:  item.GetKeyId(),
				Result: &pk.BatchUpdateKeyMetadataResult_Error{Error: err.Error()},
			})
			if !req.GetContinueOnError() {
				return nil, fmt.Errorf("invalid key ID in batch request: %w", err)
			}
			continue
		}

		// Get the current key to apply updates
		currentKey, err := s.keyRepo.GetKey(ctx, keyID)
		if err != nil {
			failedCount++
			results = append(results, &pk.BatchUpdateKeyMetadataResult{
				KeyId:  item.GetKeyId(),
				Result: &pk.BatchUpdateKeyMetadataResult_Error{Error: fmt.Sprintf("failed to get key for update: %v", err)},
			})
			if !req.GetContinueOnError() {
				return nil, fmt.Errorf("failed to get key for update: %w", err)
			}
			continue
		}

		metadata := currentKey.Metadata

		if item.Description != nil {
			description, err := domain.NewDescription(*item.Description)
			if err != nil {
				failedCount++
				results = append(results, &pk.BatchUpdateKeyMetadataResult{
					KeyId:  item.GetKeyId(),
					Result: &pk.BatchUpdateKeyMetadataResult_Error{Error: fmt.Sprintf("invalid description: %v", err)},
				})
				if !req.GetContinueOnError() {
					return nil, fmt.Errorf("invalid description: %w", err)
				}
				continue
			}
			metadata.Description = description.String()
		}
		if item.ExpiresAt != nil {
			metadata.ExpiresAt = item.ExpiresAt
		}
		if item.DataClassification != nil {
			metadata.DataClassification = *item.DataClassification
		}

		if len(item.GetTagsToAdd()) > 0 || len(item.GetTagsToRemove()) > 0 {
			if metadata.Tags == nil {
				metadata.Tags = make(map[string]string)
			}
			if len(item.GetTagsToAdd()) > 0 {
				maps.Copy(metadata.Tags, item.GetTagsToAdd())
			}
			for _, tag := range item.GetTagsToRemove() {
				delete(metadata.Tags, tag)
			}
		}

		// Policies to update
		if len(item.GetPoliciesToUpdate()) > 0 {
			if metadata.AccessPolicies == nil {
				metadata.AccessPolicies = make(map[string]string)
			}
			maps.Copy(metadata.AccessPolicies, item.GetPoliciesToUpdate())
		}

		metadata.UpdatedAt = timestamppb.Now()
		currentKey.Metadata = metadata
		currentKey.UpdatedAt = time.Now()
		keysToUpdate = append(keysToUpdate, currentKey)
	}

	// Perform batch update in repository
	if err := s.keyRepo.UpdateBatchKeyMetadata(ctx, keysToUpdate); err != nil {
		// If the entire batch update fails, return a single error unless continue_on_error is true
		if !req.GetContinueOnError() {
			return nil, fmt.Errorf("failed to update batch key metadata in repository: %w", err)
		}
		// If continue_on_error is true, mark all as failed
		for _, item := range req.GetKeys() {
			failedCount++
			results = append(results, &pk.BatchUpdateKeyMetadataResult{
				KeyId:  item.GetKeyId(),
				Result: &pk.BatchUpdateKeyMetadataResult_Error{Error: err.Error()},
			})
		}
		return &pk.BatchUpdateKeyMetadataResponse{
			Results:           results,
			ResponseTimestamp: timestamppb.Now(),
			SuccessfulCount:   successCount,
			FailedCount:       failedCount,
		}, nil
	}

	// For successful batch update, we assume all keys were updated.
	// The repository doesn't return individual success/failure for batch.
	for _, item := range req.GetKeys() {
		successCount++
		results = append(results, &pk.BatchUpdateKeyMetadataResult{
			KeyId:  item.GetKeyId(),
			Result: &pk.BatchUpdateKeyMetadataResult_Success{Success: true},
		})
	}

	return &pk.BatchUpdateKeyMetadataResponse{
		Results:           results,
		ResponseTimestamp: timestamppb.Now(),
		SuccessfulCount:   successCount,
		FailedCount:       failedCount,
	}, nil
}