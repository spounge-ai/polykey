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

	dek := make([]byte, dekSize)
	if _, err := rand.Read(dek); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKeyGenerationFail, err)
	}
	defer zeroBytes(dek)

	if err := validateEntropy(dek); err != nil {
		return nil, err
	}

	keyID := domain.NewKeyID()
	now := time.Now()

	newKey := &domain.Key{
		ID:           keyID,
		Version:      1,
		EncryptedDEK: dek, // Temporary, encrypted below
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

	kmsProvider, err := s.getKMSProvider(newKey)
	if err != nil {
		return nil, err
	}

	if newKey.EncryptedDEK, err = kmsProvider.EncryptDEK(ctx, dek, newKey); err != nil {
		return nil, fmt.Errorf("failed to encrypt DEK: %w", err)
	}

	tier := newKey.GetTier()
	if err := s.keyRepo.CreateKey(ctx, newKey, tier == domain.TierPro || tier == domain.TierEnterprise); err != nil {
		return nil, fmt.Errorf("failed to create key: %w", err)
	}

	s.logger.InfoContext(ctx, "key created", "keyId", keyID, "keyType", req.GetKeyType().String())

	return &pk.CreateKeyResponse{
		KeyId:    keyID.String(),
		Metadata: newKey.Metadata,
		KeyMaterial: &pk.KeyMaterial{
			EncryptedKeyData:    append([]byte(nil), newKey.EncryptedDEK...),
			EncryptionAlgorithm: algorithm,
			KeyChecksum:         "sha256",
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