package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/spounge-ai/polykey/internal/domain"
	app_errors "github.com/spounge-ai/polykey/internal/errors"
	"github.com/spounge-ai/polykey/pkg/authorization"
	"github.com/spounge-ai/polykey/pkg/crypto"
	"github.com/spounge-ai/polykey/pkg/patterns/batch"
	"google.golang.org/protobuf/types/known/timestamppb"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

const MaxKeyIDGenerationRetries = 10

// createKeyObject encapsulates the core logic for creating a new key domain object.
// It handles DEK generation, encryption, and metadata population.
func (s *keyServiceImpl) createKeyObject(ctx context.Context, item *pk.CreateKeyItem, clientIdentity string, storageProfile pk.StorageProfile) (*domain.Key, error) {
	description, err := domain.NewDescription(item.GetDescription())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", app_errors.ErrInvalidInput, err)
	}

	dekPool, ok := s.dekPools[item.GetKeyType()]
	if !ok {
		return nil, fmt.Errorf("%w: unsupported key type for pooling", ErrInvalidKeyType)
	}

	dek := dekPool.Get()
	defer dekPool.Put(dek)

	if _, err := rand.Read(dek); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrKeyGenerationFail, err)
	}

	keyID := domain.NewKeyID()
	now := time.Now()

	kmsProvider, err := s.getKMSProvider(storageProfile)
	if err != nil {
		return nil, err
	}

	finalKey := &domain.Key{
		ID:        keyID,
		Version:   1,
		Status:    domain.KeyStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata: &pk.KeyMetadata{
			KeyId:              keyID.String(),
			KeyType:            item.GetKeyType(),
			Status:             pk.KeyStatus_KEY_STATUS_ACTIVE,
			Version:            1,
			CreatedAt:          timestamppb.New(now),
			UpdatedAt:          timestamppb.New(now),
			ExpiresAt:          item.GetExpiresAt(),
			CreatorIdentity:    clientIdentity,
			AuthorizedContexts: item.GetInitialAuthorizedContexts(),
			AccessPolicies:     item.GetAccessPolicies(),
			Description:        description.String(),
			Tags:               item.GetTags(),
			DataClassification: item.GetDataClassification(),
			StorageType:        storageProfile,
			AccessCount:        0,
		},
	}

	encryptedDEK, err := kmsProvider.EncryptDEK(ctx, dek, finalKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt DEK: %w", err)
	}

	finalKey.EncryptedDEK = encryptedDEK
	return finalKey, nil
}

func (s *keyServiceImpl) CreateKey(ctx context.Context, req *pk.CreateKeyRequest) (*pk.CreateKeyResponse, error) {
	if req == nil || req.RequesterContext == nil || req.RequesterContext.GetClientIdentity() == "" {
		return nil, app_errors.ErrInvalidInput
	}

	authenticatedUser, _ := domain.UserFromContext(ctx)
	storageProfile := authorization.GetStorageProfileForTier(authenticatedUser.Tier)

	_, algorithm, err := crypto.GetCryptoDetails(req.GetKeyType())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", app_errors.ErrInvalidInput, err)
	}

	// Adapt the single request to the item format expected by the helper.
	item := &pk.CreateKeyItem{
		KeyType:                   req.GetKeyType(),
		Description:               req.GetDescription(),
		Tags:                      req.GetTags(),
		ExpiresAt:                 req.GetExpiresAt(),
		InitialAuthorizedContexts: req.GetInitialAuthorizedContexts(),
		AccessPolicies:            req.GetAccessPolicies(),
		DataClassification:        req.GetDataClassification(),
		GenerationParams:          req.GetGenerationParams(),
	}

	finalKey, err := s.createKeyObject(ctx, item, req.RequesterContext.GetClientIdentity(), storageProfile)
	if err != nil {
		return nil, err
	}

	createdKey, err := s.keyRepo.CreateKey(ctx, finalKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create key: %w", err)
	}

	s.logger.InfoContext(ctx, "key created", "keyId", createdKey.ID, "keyType", req.GetKeyType().String())

	return &pk.CreateKeyResponse{
		KeyId:    createdKey.ID.String(),
		Metadata: createdKey.Metadata,
		KeyMaterial: &pk.KeyMaterial{
			EncryptedKeyData:    append([]byte(nil), createdKey.EncryptedDEK...),
			EncryptionAlgorithm: algorithm,
			KeyChecksum:         "sha256", // Note: This checksum is of the *encrypted* key, which is less useful.
		},
		ResponseTimestamp: timestamppb.Now(),
	}, nil
}

func (s *keyServiceImpl) BatchCreateKeys(ctx context.Context, req *pk.BatchCreateKeysRequest) (*pk.BatchCreateKeysResponse, error) {
	if req == nil || req.RequesterContext == nil || req.RequesterContext.GetClientIdentity() == "" {
		return nil, app_errors.ErrInvalidInput
	}

	authenticatedUser, _ := domain.UserFromContext(ctx)
	storageProfile := authorization.GetStorageProfileForTier(authenticatedUser.Tier)

	processor := batch.BatchProcessor[*pk.CreateKeyItem, *domain.Key]{
		MaxConcurrency: 10, // Make this configurable
		Validate: func(item *pk.CreateKeyItem) error {
			if _, _, err := crypto.GetCryptoDetails(item.GetKeyType()); err != nil {
				return fmt.Errorf("%w: %w", app_errors.ErrInvalidInput, err)
			}
			if _, err := domain.NewDescription(item.GetDescription()); err != nil {
				return fmt.Errorf("%w: %w", app_errors.ErrInvalidInput, err)
			}
			return nil
		},
		Process: func(ctx context.Context, item *pk.CreateKeyItem) (*domain.Key, error) {
			return s.createKeyObject(ctx, item, req.RequesterContext.GetClientIdentity(), storageProfile)
		},
	}

	results, err := processor.ProcessBatch(ctx, req.Keys, req.GetContinueOnError())
	if err != nil {
		return nil, err
	}

	createdKeys := make([]*domain.Key, 0, len(results.Items))
	batchResults := make([]*pk.BatchCreateKeysResult, len(results.Items))
	for i, item := range results.Items {
		if item.Error != nil {
			batchResults[i] = &pk.BatchCreateKeysResult{
				Result: &pk.BatchCreateKeysResult_Error{Error: item.Error.Error()},
			}
		} else {
			createdKeys = append(createdKeys, item.Result)
			batchResults[i] = &pk.BatchCreateKeysResult{
				Result: &pk.BatchCreateKeysResult_Success{
					Success: &pk.CreateKeyResponse{
						KeyId:    item.Result.ID.String(),
						Metadata: item.Result.Metadata,
					},
				},
			}
		}
	}

	if err := s.keyRepo.CreateBatchKeys(ctx, createdKeys); err != nil {
		return nil, fmt.Errorf("failed to create keys in batch: %w", err)
	}

	return &pk.BatchCreateKeysResponse{
		Results: batchResults,
	}, nil
}