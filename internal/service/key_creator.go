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
		s.logger.WarnContext(ctx, "invalid create key request")
		return nil, fmt.Errorf("%w: requester context required", ErrInvalidRequest)
	}

	description, err := domain.NewDescription(req.GetDescription())
	if err != nil {
		return nil, err
	}

	dekSize, algorithm, err := getCryptoDetails(req.GetKeyType())
	if err != nil {
		s.logger.ErrorContext(ctx, "unsupported key type", "keyType", req.GetKeyType())
		return nil, err
	}

	dek := make([]byte, dekSize)
	if _, err := rand.Read(dek); err != nil {
		s.logger.ErrorContext(ctx, "failed to generate DEK", "error", err)
		return nil, fmt.Errorf("%w: %v", ErrKeyGenerationFail, err)
	}

	if err := validateEntropy(dek); err != nil {
		return nil, err
	}

	keyID := domain.NewKeyID()
	now := time.Now()

	metadata := &pk.KeyMetadata{
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
	}

	newKey := &domain.Key{
		ID:           keyID,
		Version:      1,
		Metadata:     metadata,
		EncryptedDEK: dek, // Temporary, will be encrypted below
		Status:       domain.KeyStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	kmsProvider, err := s.getKMSProvider(newKey)
	if err != nil {
		return nil, err
	}

	encryptedDEK, err := kmsProvider.EncryptDEK(ctx, newKey)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to encrypt DEK", "error", err)
		return nil, fmt.Errorf("failed to encrypt DEK: %w", err)
	}
	newKey.EncryptedDEK = encryptedDEK

	tier := newKey.GetTier()
	if err := s.keyRepo.CreateKey(ctx, newKey, tier == domain.TierPro || tier == domain.TierEnterprise); err != nil {
		s.logger.ErrorContext(ctx, "failed to create key", "keyId", keyID, "error", err)
		return nil, fmt.Errorf("failed to create key: %w", err)
	}

	resp := &pk.CreateKeyResponse{
		KeyId:    keyID.String(),
		Metadata: metadata,
		KeyMaterial: &pk.KeyMaterial{
			EncryptedKeyData:    append([]byte(nil), encryptedDEK...),
			EncryptionAlgorithm: algorithm,
			KeyChecksum:         "sha256",
		},
		ResponseTimestamp: timestamppb.Now(),
	}

	s.logger.InfoContext(ctx, "key created", "keyId", keyID, "keyType", req.GetKeyType().String(), "creator", req.RequesterContext.GetClientIdentity())
	return resp, nil
}

func validateEntropy(data []byte) error {
	// A simple entropy check: the number of set bits should be roughly half the total number of bits.
	// This is not a perfect entropy test, but it can catch some basic generation failures.
	setBits := 0
	for _, b := range data {
		setBits += bits.OnesCount8(b)
	}

	totalBits := len(data) * 8
	// The number of set bits should be between 37.5% and 62.5% of the total bits.
	// These bounds are arbitrary and can be adjusted.
	lowerBound := totalBits * 3 / 8
	upperBound := totalBits * 5 / 8

	if setBits < lowerBound || setBits > upperBound {
		return fmt.Errorf("%w: expected between %d and %d set bits, but got %d", ErrEntropyValidationFail, lowerBound, upperBound, setBits)
	}

	return nil
}
