package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/spounge-ai/polykey/internal/domain"
	app_errors "github.com/spounge-ai/polykey/internal/errors"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *keyServiceImpl) CreateKey(ctx context.Context, req *pk.CreateKeyRequest) (*pk.CreateKeyResponse, error) {
	if req == nil || req.RequesterContext == nil || req.RequesterContext.GetClientIdentity() == "" {
		return nil, app_errors.ErrInvalidInput
	}

	authenticatedUser, ok := domain.UserFromContext(ctx)
	if !ok {
		return nil, app_errors.ErrAuthentication
	}

	if err := validateTier(authenticatedUser.Tier, req.GetDataClassification()); err != nil {
		return nil, fmt.Errorf("%w: %w", app_errors.ErrAuthorization, err)
	}

	description, err := domain.NewDescription(req.GetDescription())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", app_errors.ErrInvalidInput, err)
	}

	dekSize, algorithm, err := getCryptoDetails(req.GetKeyType())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", app_errors.ErrInvalidInput, err)
	}

	// Generate the DEK and ensure it is securely zeroed after use.
	dek := make([]byte, dekSize)
	if _, err := rand.Read(dek); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrKeyGenerationFail, err)
	}
	defer secureZeroBytes(dek)

	// Generate a unique KeyID, retrying up to 10 times in the unlikely event of a collision.
	var keyID domain.KeyID
	for i := 0; i < 10; i++ {
		keyID = domain.NewKeyID()
		exists, err := s.keyRepo.Exists(ctx, keyID)
		if err != nil {
			return nil, fmt.Errorf("failed to check key existence: %w", err)
		}
		if !exists {
			break
		}
		if i == 9 {
			return nil, fmt.Errorf("failed to generate a unique key ID after 10 attempts")
		}
	}

	now := time.Now()

	kmsProvider, err := s.getKMSProvider(req.GetDataClassification())
	if err != nil {
		return nil, err
	}

	// Create a temporary key object to pass to the KMS provider if needed.
	tempKeyForKMS := &domain.Key{}

	// Immediate encryption pattern: Encrypt the DEK right after generation.
	encryptedDEK, err := kmsProvider.EncryptDEK(ctx, dek, tempKeyForKMS)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt DEK: %w", err)
	}

	// Now, populate the final domain object with the encrypted DEK for storage.
	finalKey := &domain.Key{
		ID:           keyID,
		Version:      1,
		EncryptedDEK: encryptedDEK,
		Status:       domain.KeyStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
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
			AccessCount:        0,
		},
	}

	// The key's tier is now based on the validated data classification.
	isPremium := req.GetDataClassification() == string(domain.TierPro) || req.GetDataClassification() == string(domain.TierEnterprise)
	if err := s.keyRepo.CreateKey(ctx, finalKey, isPremium); err != nil {
		return nil, fmt.Errorf("failed to create key: %w", err)
	}

	s.logger.InfoContext(ctx, "key created", "keyId", keyID, "keyType", req.GetKeyType().String())

	return &pk.CreateKeyResponse{
		KeyId:    keyID.String(),
		Metadata: finalKey.Metadata,
		KeyMaterial: &pk.KeyMaterial{
			EncryptedKeyData:    append([]byte(nil), finalKey.EncryptedDEK...),
			EncryptionAlgorithm: algorithm,
			KeyChecksum:         "sha256", // Note: This checksum is of the *encrypted* key, which is less useful.
		},
		ResponseTimestamp: timestamppb.Now(),
	}, nil
}

func validateTier(clientTier domain.KeyTier, requestedClassification string) error {
	switch clientTier {
	case domain.TierEnterprise:
		// Enterprise clients can create keys of any classification.
		return nil
	case domain.TierPro:
		if requestedClassification == string(domain.TierEnterprise) {
			return fmt.Errorf("pro tier clients cannot create enterprise classification keys")
		}
		return nil
	case domain.TierFree:
		if requestedClassification == string(domain.TierEnterprise) || requestedClassification == string(domain.TierPro) {
			return fmt.Errorf("free tier clients can only create free classification keys")
		}
		return nil
	default:
		// Default to free tier behavior if client tier is unknown or not set.
		if requestedClassification == string(domain.TierEnterprise) || requestedClassification == string(domain.TierPro) {
			return fmt.Errorf("clients with no tier can only create free classification keys")
		}
		return nil
	}
}


