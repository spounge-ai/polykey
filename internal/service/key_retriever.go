package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/spounge-ai/polykey/internal/domain"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *keyServiceImpl) GetKey(ctx context.Context, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: request is nil", ErrInvalidRequest)
	}
	keyID, err := domain.KeyIDFromString(req.GetKeyId())
	if err != nil {
		return nil, err
	}

	key, err := s.getKeyByRequest(ctx, keyID, req.GetVersion())
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get key", "keyId", req.GetKeyId(), "error", err)
		return nil, fmt.Errorf("failed to get key: %w", err)
	}

	if key.Metadata == nil {
		return nil, ErrMissingMetadata
	}

	kmsProvider, err := s.getKMSProvider(key.Metadata.GetDataClassification())
	if err != nil {
		return nil, fmt.Errorf("failed to get KMS provider: %w", err)
	}

	decryptedDEK, err := kmsProvider.DecryptDEK(ctx, key)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to decrypt DEK", "keyId", req.GetKeyId(), "error", err)
		return nil, fmt.Errorf("failed to decrypt DEK: %w", err)
	}
	defer secureZeroBytes(decryptedDEK)

	_, algorithm, err := getCryptoDetails(key.Metadata.GetKeyType())
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get crypto details for key type", "keyId", req.GetKeyId(), "keyType", key.Metadata.GetKeyType(), "error", err)
		algorithm = "unknown"
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

	s.logger.InfoContext(ctx, "key retrieved and decrypted", "keyId", req.GetKeyId(), "version", key.Version)
	return resp, nil
}

func (s *keyServiceImpl) GetKeyMetadata(ctx context.Context, req *pk.GetKeyMetadataRequest) (*pk.GetKeyMetadataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: request is nil", ErrInvalidRequest)
	}
	keyID, err := domain.KeyIDFromString(req.GetKeyId())
	if err != nil {
		return nil, err
	}

	key, err := s.getKeyByRequest(ctx, keyID, req.GetVersion())
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get key metadata", "keyId", req.GetKeyId(), "error", err)
		return nil, fmt.Errorf("failed to get key metadata: %w", err)
	}

	resp := &pk.GetKeyMetadataResponse{
		Metadata:          key.Metadata,
		ResponseTimestamp: timestamppb.Now(),
	}

	if req.GetIncludeAccessHistory() {
		s.logger.WarnContext(ctx, "IncludeAccessHistory not implemented", "keyId", req.GetKeyId())
	}
	if req.GetIncludePolicyDetails() {
		s.logger.WarnContext(ctx, "IncludePolicyDetails not implemented", "keyId", req.GetKeyId())
	}

	s.logger.InfoContext(ctx, "key metadata retrieved", "keyId", req.GetKeyId(), "version", key.Version)
	return resp, nil
}
