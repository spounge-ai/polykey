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

func (s *keyServiceImpl) BatchRotateKeys(ctx context.Context, req *pk.BatchRotateKeysRequest) (*pk.BatchRotateKeysResponse, error) {
	ctx, span := tracer.Start(ctx, "BatchRotateKeys")
	defer span.End()

	if req == nil || req.RequesterContext == nil || req.RequesterContext.GetClientIdentity() == "" {
		return nil, ErrInvalidRequest
	}

	var (
		successCount int32
		failedCount  int32
		results      []*pk.BatchRotateKeysResult
	)

	// Use a channel to collect results from the pipeline workers
	batchResults := make(chan pipelines.KeyRotationResult, len(req.GetKeys()))

	for _, item := range req.GetKeys() {
		keyID, err := domain.KeyIDFromString(item.GetKeyId())
		if err != nil {
			failedCount++
			results = append(results, &pk.BatchRotateKeysResult{
				KeyId: item.GetKeyId(),
				Result: &pk.BatchRotateKeysResult_Error{Error: err.Error()},
			})
			if !req.GetContinueOnError() {
				return nil, fmt.Errorf("invalid key ID in batch request: %w", err)
			}
			continue
		}

		// Get the current key to determine the storage profile and DEK pool
		currentKey, err := s.keyRepo.GetKey(ctx, keyID)
		if err != nil {
			failedCount++
			results = append(results, &pk.BatchRotateKeysResult{
				KeyId: item.GetKeyId(),
				Result: &pk.BatchRotateKeysResult_Error{Error: fmt.Sprintf("failed to get current key: %v", err)},
			})
			if !req.GetContinueOnError() {
				return nil, fmt.Errorf("failed to get current key for rotation: %w", err)
			}
			continue
		}

		kmsProvider, err := s.getKMSProvider(currentKey.Metadata.GetStorageType())
		if err != nil {
			failedCount++
			results = append(results, &pk.BatchRotateKeysResult{
				KeyId: item.GetKeyId(),
				Result: &pk.BatchRotateKeysResult_Error{Error: fmt.Sprintf("failed to get KMS provider: %v", err)},
			})
			if !req.GetContinueOnError() {
				return nil, fmt.Errorf("failed to get KMS provider for rotation: %w", err)
			}
			continue
		}

		dekPool, ok := s.dekPools[currentKey.Metadata.GetKeyType()]
		if !ok {
			failedCount++
			results = append(results, &pk.BatchRotateKeysResult{
				KeyId: item.GetKeyId(),
				Result: &pk.BatchRotateKeysResult_Error{Error: fmt.Sprintf("unsupported key type for pooling: %v", currentKey.Metadata.GetKeyType())},
			})
			if !req.GetContinueOnError() {
				return nil, fmt.Errorf("unsupported key type for pooling: %w", ErrInvalidKeyType)
			}
			continue
		}

		rotationReq := pipelines.KeyRotationRequest{
			KeyID:       keyID,
			KMSProvider: kmsProvider,
			DEKPool:     dekPool,
		}

		if !s.keyRotationPipeline.Enqueue(rotationReq) {
			failedCount++
			results = append(results, &pk.BatchRotateKeysResult{
				KeyId: item.GetKeyId(),
				Result: &pk.BatchRotateKeysResult_Error{Error: "key rotation queue is full"},
			})
			if !req.GetContinueOnError() {
				return nil, status.Errorf(codes.ResourceExhausted, "key rotation queue is full, please try again later")
			}
			continue
		}
		// If enqueued, we expect a result back on the channel
		go func(keyID domain.KeyID) {
			select {
			case result := <-s.keyRotationPipeline.Results():
				batchResults <- result
			case <-ctx.Done():
				// If context is cancelled, send an error result
				batchResults <- pipelines.KeyRotationResult{RotatedKey: nil, Error: ctx.Err()}
			}
		}(keyID)
	}

	// Collect results from the pipeline
	for i := 0; i < len(req.GetKeys()); i++ {
		select {
		case result := <-batchResults:
			if result.Error != nil {
				failedCount++
				results = append(results, &pk.BatchRotateKeysResult{
					KeyId: result.RotatedKey.ID.String(), // Assuming ID is available even on error
					Result: &pk.BatchRotateKeysResult_Error{Error: result.Error.Error()},
				})
			} else {
				successCount++
				gracePeriod := time.Duration(req.GetKeys()[i].GetGracePeriodSeconds()) * time.Second // This is problematic, need to map back to original request item
				now := time.Now()
				resp := &pk.RotateKeyResponse{
					KeyId:           result.RotatedKey.ID.String(),
					NewVersion:      result.RotatedKey.Version,
					PreviousVersion: result.RotatedKey.Version - 1, // Assuming version increments by 1
					NewKeyMaterial: &pk.KeyMaterial{
						EncryptedKeyData:    append([]byte(nil), result.RotatedKey.EncryptedDEK...),
						EncryptionAlgorithm: "AES-256-GCM", // This should be dynamic based on key type
						KeyChecksum:         "sha256",
					},
					Metadata:            result.RotatedKey.Metadata,
					RotationTimestamp:   timestamppb.New(now),
					OldVersionExpiresAt: timestamppb.New(now.Add(gracePeriod)),
				}
				results = append(results, &pk.BatchRotateKeysResult{
					KeyId: result.RotatedKey.ID.String(),
					Result: &pk.BatchRotateKeysResult_Success{Success: resp},
				})
			}
		case <-ctx.Done():
			// If context is cancelled while collecting results, stop and return current results
			return &pk.BatchRotateKeysResponse{
				Results:           results,
				ResponseTimestamp: timestamppb.Now(),
				SuccessfulCount:   successCount,
				FailedCount:       failedCount + (int32(len(req.GetKeys())) - successCount - failedCount), // Mark remaining as failed
			}, ctx.Err()
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

	keyIDs := make([]domain.KeyID, len(req.GetKeys()))
	for _, item := range req.GetKeys() {
		keyID, err := domain.KeyIDFromString(item.GetKeyId())
		if err != nil {
			failedCount++
			results = append(results, &pk.BatchRevokeKeysResult{
				KeyId: item.GetKeyId(),
				Result: &pk.BatchRevokeKeysResult_Error{Error: err.Error()},
			})
			if !req.GetContinueOnError() {
				return nil, fmt.Errorf("invalid key ID in batch request: %w", err)
			}
			continue
		}
		keyIDs = append(keyIDs, keyID)
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
				KeyId: id.String(),
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
			KeyId: id.String(),
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
				KeyId: item.GetKeyId(),
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
				KeyId: item.GetKeyId(),
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
					KeyId: item.GetKeyId(),
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
				KeyId: item.GetKeyId(),
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
			KeyId: item.GetKeyId(),
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
