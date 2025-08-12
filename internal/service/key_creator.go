package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/bits"
	"time"

	"github.com/spounge-ai/polykey/internal/domain"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *keyServiceImpl) CreateKey(ctx context.Context, req *pk.CreateKeyRequest) (*pk.CreateKeyResponse, error) {
	if req == nil || req.RequesterContext == nil || req.RequesterContext.GetClientIdentity() == "" {
		return nil, fmt.Errorf("%w: requester context required", ErrInvalidRequest)
	}

	description, err := domain.NewDescription(req.GetDescription())
	if err != nil {
		return nil, err
	}

	dekSize, algorithm, err := getCryptoDetails(req.GetKeyType())
	if err != nil {
		return nil, err
	}

	// Generate the DEK and ensure it is securely zeroed after use.
	dek := make([]byte, dekSize)
	if _, err := rand.Read(dek); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKeyGenerationFail, err)
	}
	defer secureZeroBytes(dek)

	if err := validateEntropy(dek); err != nil {
		return nil, err
	}

	keyID := domain.NewKeyID()
	now := time.Now()

	// The domain.Key object is created here to determine the tier and select the correct KMS provider.
	// The EncryptedDEK field is intentionally left nil initially.
	keyToCreate := &domain.Key{
		ID:      keyID,
		Version: 1,
		Status:  domain.KeyStatusActive,
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

	kmsProvider, err := s.getKMSProvider(keyToCreate)
	if err != nil {
		return nil, err
	}

	// Immediate encryption pattern: Encrypt the DEK right after generation.
	encryptedDEK, err := kmsProvider.EncryptDEK(ctx, dek, keyToCreate)
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
		Metadata:     keyToCreate.Metadata,
	}

	tier := finalKey.GetTier()
	if err := s.keyRepo.CreateKey(ctx, finalKey, tier == domain.TierPro || tier == domain.TierEnterprise); err != nil {
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

func validateEntropy(data []byte) error {
	setBits := 0
	for _, b := range data {
		setBits += bits.OnesCount8(b)
	}

	totalBits := len(data) * 8
	lowerBound := totalBits * 3 / 8  // 37.5%
	upperBound := totalBits * 5 / 8  // 62.5%

	if setBits < lowerBound || setBits > upperBound {
		return fmt.Errorf("%w: expected %d-%d set bits, got %d", 
			ErrEntropyValidationFail, lowerBound, upperBound, setBits)
	}
	return nil
}