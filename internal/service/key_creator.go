package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/spounge-ai/polykey/internal/domain"
	app_errors "github.com/spounge-ai/polykey/internal/errors"
	"github.com/spounge-ai/polykey/pkg/crypto"
	"google.golang.org/protobuf/types/known/timestamppb"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2" 
)

const MaxKeyIDGenerationRetries = 10

func (s *keyServiceImpl) CreateKey(ctx context.Context, req *pk.CreateKeyRequest) (*pk.CreateKeyResponse, error) {
	if req == nil || req.RequesterContext == nil || req.RequesterContext.GetClientIdentity() == "" {
		return nil, app_errors.ErrInvalidInput
	}

	authenticatedUser, ok := domain.UserFromContext(ctx)
	if !ok {
		return nil, app_errors.ErrAuthentication
	}

	storageProfile := pk.StorageProfile_STORAGE_PROFILE_STANDARD
	if authenticatedUser.Tier == domain.TierPro || authenticatedUser.Tier == domain.TierEnterprise {
		storageProfile = pk.StorageProfile_STORAGE_PROFILE_HARDENED
	}

	description, err := domain.NewDescription(req.GetDescription())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", app_errors.ErrInvalidInput, err)
	}

	_, algorithm, err := crypto.GetCryptoDetails(req.GetKeyType())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", app_errors.ErrInvalidInput, err)
	}

	dekPool, ok := s.dekPools[req.GetKeyType()]
	if !ok {
		return nil, fmt.Errorf("%w: unsupported key type for pooling", ErrInvalidKeyType)
	}

	// Generate the DEK from a secure pool and ensure it is returned.
	dek := dekPool.Get()
	defer dekPool.Put(dek)

	if _, err := rand.Read(dek); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrKeyGenerationFail, err)
	}

	// Generate a unique KeyID.
	keyID := domain.NewKeyID()

	now := time.Now()

	kmsProvider, err := s.getKMSProvider(storageProfile)
	if err != nil {
		return nil, err
	}

	// Create the final key object first, so we have the ID for the KDF.
	finalKey := &domain.Key{
		ID:        keyID,
		Version:   1,
		Status:    domain.KeyStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata: &pk.KeyMetadata{
			KeyId:              keyID.String(),
			KeyType:            req.GetKeyType(),
			Status:             pk.KeyStatus_KEY_STATUS_ACTIVE,
			Version:            1,
			CreatedAt:          timestamppb.New(now),
			UpdatedAt:          timestamppb.New(now),
			ExpiresAt:          req.GetExpiresAt(),
			CreatorIdentity:    req.RequesterContext.GetClientIdentity(),
			AuthorizedContexts: req.GetInitialAuthorizedContexts(),
			AccessPolicies:     req.GetAccessPolicies(),
			Description:        description.String(),
			Tags:               req.GetTags(),
			DataClassification: req.GetDataClassification(),
			StorageType:        storageProfile,
			AccessCount:        0,
		},
	}

	// Immediate encryption pattern: Encrypt the DEK right after generation.
	encryptedDEK, err := kmsProvider.EncryptDEK(ctx, dek, finalKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt DEK: %w", err)
	}

	// Now, populate the final domain object with the encrypted DEK for storage.
	finalKey.EncryptedDEK = encryptedDEK

	createdKey, err := s.keyRepo.CreateKey(ctx, finalKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create key: %w", err)
	}

	s.logger.InfoContext(ctx, "key created", "keyId", keyID, "keyType", req.GetKeyType().String())

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




