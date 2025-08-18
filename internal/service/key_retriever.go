package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/spounge-ai/polykey/internal/domain"
	app_errors "github.com/spounge-ai/polykey/internal/errors"
	"github.com/spounge-ai/polykey/pkg/crypto"
	"github.com/spounge-ai/polykey/pkg/memory"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var tracer = otel.Tracer("github.com/spounge-ai/polykey/internal/service")

func (s *keyServiceImpl) GetKey(ctx context.Context, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error) {
	ctx, span := tracer.Start(ctx, "GetKey")
	defer span.End()

	if req == nil {
		return nil, app_errors.ErrInvalidInput
	}
	keyID, err := domain.KeyIDFromString(req.GetKeyId())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", app_errors.ErrInvalidInput, err)
	}

	span.SetAttributes(attribute.String("key.id", keyID.String()))

	key, err := s.getKeyByRequest(ctx, keyID, req.GetVersion())
	if err != nil {
		return nil, err
	}

	if key.Metadata == nil {
		return nil, ErrMissingMetadata
	}

	kmsProvider, err := s.getKMSProvider(key.Metadata.GetStorageType())
	if err != nil {
		return nil, fmt.Errorf("failed to get KMS provider: %w", err)
	}

	decryptedDEK, err := kmsProvider.DecryptDEK(ctx, key)
	if err != nil {
		s.auditLogger.AuditLog(ctx, req.GetRequesterContext().GetClientIdentity(), "GetKey", keyID.String(), "", false, err)
		return nil, fmt.Errorf("%w: %w", app_errors.ErrKMSFailure, err)
	}
	defer memory.SecureZeroBytes(decryptedDEK)

	_, algorithm, err := crypto.GetCryptoDetails(key.Metadata.GetKeyType())
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256(decryptedDEK)
	checksum := hex.EncodeToString(hash[:])

	resp := &pk.GetKeyResponse{
		KeyMaterial: &pk.KeyMaterial{
			EncryptedKeyData:    key.EncryptedDEK,
			EncryptionAlgorithm: algorithm,
			KeyChecksum:         checksum,
		},
		ResponseTimestamp: timestamppb.Now(),
	}

	if !req.GetSkipMetadata() {
		resp.Metadata = key.Metadata
	}

	s.auditLogger.AuditLog(ctx, req.GetRequesterContext().GetClientIdentity(), "GetKey", keyID.String(), "", true, nil)
	s.logger.InfoContext(ctx, "key retrieved and decrypted", "keyId", req.GetKeyId(), "version", key.Version)
	return resp, nil
}

func (s *keyServiceImpl) GetKeyMetadata(ctx context.Context, req *pk.GetKeyMetadataRequest) (*pk.GetKeyMetadataResponse, error) {
	ctx, span := tracer.Start(ctx, "GetKeyMetadata")
	defer span.End()

	if req == nil {
		return nil, app_errors.ErrInvalidInput
	}
	keyID, err := domain.KeyIDFromString(req.GetKeyId())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", app_errors.ErrInvalidInput, err)
	}

	span.SetAttributes(attribute.String("key.id", keyID.String()))

	var metadata *pk.KeyMetadata
	if req.GetVersion() > 0 {
		metadata, err = s.keyRepo.GetKeyMetadataByVersion(ctx, keyID, req.GetVersion())
	} else {
		metadata, err = s.keyRepo.GetKeyMetadata(ctx, keyID)
	}

	if err != nil {
		s.logger.ErrorContext(ctx, "[key_retriever.go:GetKeyMetadata] Error from keyRepo", "error", err)
		return nil, err
	}

	resp := &pk.GetKeyMetadataResponse{
		Metadata:          metadata,
		ResponseTimestamp: timestamppb.Now(),
	}

	if req.GetIncludeAccessHistory() {
		s.logger.WarnContext(ctx, "IncludeAccessHistory not implemented", "keyId", req.GetKeyId())
	}
	if req.GetIncludePolicyDetails() {
		s.logger.WarnContext(ctx, "IncludePolicyDetails not implemented", "keyId", req.GetKeyId())
	}

	s.auditLogger.AuditLog(ctx, req.GetRequesterContext().GetClientIdentity(), "GetKeyMetadata", keyID.String(), "", true, nil)
	s.logger.InfoContext(ctx, "key metadata retrieved", "keyId", req.GetKeyId(), "version", metadata.Version)
	return resp, nil
}

func (s *keyServiceImpl) BatchGetKeys(ctx context.Context, req *pk.BatchGetKeysRequest) (*pk.BatchGetKeysResponse, error) {
	ctx, span := tracer.Start(ctx, "BatchGetKeys")
	defer span.End()

	if req == nil || req.RequesterContext == nil || req.RequesterContext.GetClientIdentity() == "" {
		return nil, app_errors.ErrInvalidInput
	}

	var ( 
		successCount int32
		failedCount  int32
		results      []*pk.BatchGetKeysResult
	)

	keyIDs := make([]domain.KeyID, len(req.GetKeys()))
	for i, item := range req.GetKeys() {
		keyID, err := domain.KeyIDFromString(item.GetKeyId())
		if err != nil {
			failedCount++
			results = append(results, &pk.BatchGetKeysResult{
				KeyId: item.GetKeyId(),
				Result: &pk.BatchGetKeysResult_Error{Error: err.Error()},
			})
			if !req.GetContinueOnError() {
				return nil, fmt.Errorf("invalid key ID in batch request: %w", err)
			}
			continue
		}
		keyIDs[i] = keyID
	}

	// Fetch all keys from the repository in a batch
	keys, err := s.keyRepo.GetBatchKeys(ctx, keyIDs)
	if err != nil {
		// If the entire batch fetch fails, return a single error unless continue_on_error is true
		if !req.GetContinueOnError() {
			return nil, fmt.Errorf("failed to retrieve batch keys from repository: %w", err)
		}
		// If continue_on_error is true, mark all as failed
		for _, id := range keyIDs {
			failedCount++
			results = append(results, &pk.BatchGetKeysResult{
				KeyId: id.String(),
				Result: &pk.BatchGetKeysResult_Error{Error: err.Error()},
			})
		}
		return &pk.BatchGetKeysResponse{
			Results:           results,
			ResponseTimestamp: timestamppb.Now(),
			SuccessfulCount:   successCount,
			FailedCount:       failedCount,
		}, nil
	}

	// Map results back to the original request order and handle individual errors
	keyMap := make(map[string]*domain.Key)
	for _, key := range keys {
		keyMap[key.ID.String()] = key
	}

	for _, item := range req.GetKeys() {
		keyIDStr := item.GetKeyId()
		if key, ok := keyMap[keyIDStr]; ok {
			// Key found, now decrypt and construct response
			kmsProvider, err := s.getKMSProvider(key.Metadata.GetStorageType())
			if err != nil {
				failedCount++
				results = append(results, &pk.BatchGetKeysResult{
					KeyId: keyIDStr,
					Result: &pk.BatchGetKeysResult_Error{Error: fmt.Sprintf("failed to get KMS provider: %v", err)},
				})
				continue
			}

			decryptedDEK, err := kmsProvider.DecryptDEK(ctx, key)
			if err != nil {
				failedCount++
				s.auditLogger.AuditLog(ctx, req.GetRequesterContext().GetClientIdentity(), "BatchGetKeys", keyIDStr, "", false, err)
				results = append(results, &pk.BatchGetKeysResult{
					KeyId: keyIDStr,
					Result: &pk.BatchGetKeysResult_Error{Error: fmt.Sprintf("KMS decryption failed: %v", err)},
				})
				continue
			}
			defer memory.SecureZeroBytes(decryptedDEK)

			_, algorithm, err := crypto.GetCryptoDetails(key.Metadata.GetKeyType())
			if err != nil {
				failedCount++
				results = append(results, &pk.BatchGetKeysResult{
					KeyId: keyIDStr,
					Result: &pk.BatchGetKeysResult_Error{Error: fmt.Sprintf("failed to get crypto details: %v", err)},
				})
				continue
			}

			hash := sha256.Sum256(decryptedDEK)
			checksum := hex.EncodeToString(hash[:])

			resp := &pk.GetKeyResponse{
				KeyMaterial: &pk.KeyMaterial{
					EncryptedKeyData:    key.EncryptedDEK,
					EncryptionAlgorithm: algorithm,
					KeyChecksum:         checksum,
				},
				ResponseTimestamp: timestamppb.Now(),
			}

			if !item.GetSkipMetadata() {
				resp.Metadata = key.Metadata
			}
			s.auditLogger.AuditLog(ctx, req.GetRequesterContext().GetClientIdentity(), "BatchGetKeys", keyIDStr, "", true, nil)
			successCount++
			results = append(results, &pk.BatchGetKeysResult{
				KeyId: keyIDStr,
				Result: &pk.BatchGetKeysResult_Success{Success: resp},
			})
		} else {
			// Key not found in the batch result from repo (e.g., due to previous error or not existing)
			failedCount++
			results = append(results, &pk.BatchGetKeysResult{
				KeyId: keyIDStr,
				Result: &pk.BatchGetKeysResult_Error{Error: "key not found or could not be processed"},
			})
		}
	}

	return &pk.BatchGetKeysResponse{
		Results:           results,
		ResponseTimestamp: timestamppb.Now(),
		SuccessfulCount:   successCount,
		FailedCount:       failedCount,
	}, nil
}

func (s *keyServiceImpl) BatchGetKeyMetadata(ctx context.Context, req *pk.BatchGetKeyMetadataRequest) (*pk.BatchGetKeyMetadataResponse, error) {
	ctx, span := tracer.Start(ctx, "BatchGetKeyMetadata")
	defer span.End()

	if req == nil || req.RequesterContext == nil || req.RequesterContext.GetClientIdentity() == "" {
		return nil, app_errors.ErrInvalidInput
	}

	var (
		successCount int32
		failedCount  int32
		results      []*pk.BatchGetKeyMetadataResult
	)

	keyIDs := make([]domain.KeyID, len(req.GetKeys()))
	for i, item := range req.GetKeys() {
		keyID, err := domain.KeyIDFromString(item.GetKeyId())
		if err != nil {
			failedCount++
			results = append(results, &pk.BatchGetKeyMetadataResult{
				KeyId: item.GetKeyId(),
				Result: &pk.BatchGetKeyMetadataResult_Error{Error: err.Error()},
			})
			if !req.GetContinueOnError() {
				return nil, fmt.Errorf("invalid key ID in batch request: %w", err)
			}
			continue
		}
		keyIDs[i] = keyID
	}

	// Fetch all key metadata from the repository in a batch
	metadataList, err := s.keyRepo.GetBatchKeyMetadata(ctx, keyIDs)
	if err != nil {
		// If the entire batch fetch fails, return a single error unless continue_on_error is true
		if !req.GetContinueOnError() {
			return nil, fmt.Errorf("failed to retrieve batch key metadata from repository: %w", err)
		}
		// If continue_on_error is true, mark all as failed
		for _, id := range keyIDs {
			failedCount++
			results = append(results, &pk.BatchGetKeyMetadataResult{
				KeyId: id.String(),
				Result: &pk.BatchGetKeyMetadataResult_Error{Error: err.Error()},
			})
		}
		return &pk.BatchGetKeyMetadataResponse{
			Results:           results,
			ResponseTimestamp: timestamppb.Now(),
			SuccessfulCount:   successCount,
			FailedCount:       failedCount,
		}, nil
	}

	// Map results back to the original request order and handle individual errors
	metadataMap := make(map[string]*pk.KeyMetadata)
	for _, md := range metadataList {
		metadataMap[md.KeyId] = md
	}

	for _, item := range req.GetKeys() {
		keyIDStr := item.GetKeyId()
		if metadata, ok := metadataMap[keyIDStr]; ok {
			successCount++
			results = append(results, &pk.BatchGetKeyMetadataResult{
				KeyId: keyIDStr,
				Result: &pk.BatchGetKeyMetadataResult_Success{Success: &pk.GetKeyMetadataResponse{Metadata: metadata, ResponseTimestamp: timestamppb.Now()}},
			})
		} else {
			// Key not found in the batch result from repo (e.g., due to previous error or not existing)
			failedCount++
			results = append(results, &pk.BatchGetKeyMetadataResult{
				KeyId: keyIDStr,
				Result: &pk.BatchGetKeyMetadataResult_Error{Error: "key metadata not found or could not be processed"},
			})
		}
	}

	return &pk.BatchGetKeyMetadataResponse{
		Results:           results,
		ResponseTimestamp: timestamppb.Now(),
		SuccessfulCount:   successCount,
		FailedCount:       failedCount,
	}, nil
}
