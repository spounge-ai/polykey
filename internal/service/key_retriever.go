package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/spounge-ai/polykey/internal/domain"
	app_errors "github.com/spounge-ai/polykey/internal/errors"
	"github.com/spounge-ai/polykey/internal/infra/persistence"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *keyServiceImpl) GetKey(ctx context.Context, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error) {
	if req == nil {
		return nil, app_errors.ErrInvalidInput
	}
	keyID, err := domain.KeyIDFromString(req.GetKeyId())
	if err != nil {
		classifiedErr := s.errorClassifier.Classify(fmt.Errorf("%w: %w", app_errors.ErrInvalidInput, err), "GetKey")
		classifiedErr.Metadata["requested_id"] = req.GetKeyId()
		return nil, s.errorClassifier.LogAndSanitize(ctx, classifiedErr)
	}

	key, err := s.getKeyByRequest(ctx, keyID, req.GetVersion())
	if err != nil {
		// The persistence layer should return a standard error type
		classifiedErr := s.errorClassifier.Classify(err, "GetKey")
		classifiedErr.KeyID = keyID.String()
		return nil, s.errorClassifier.LogAndSanitize(ctx, classifiedErr)
	}

	if key.Metadata == nil {
		return nil, ErrMissingMetadata
	}

	kmsProvider, err := s.getKMSProvider(key.Metadata.GetDataClassification())
	if err != nil {
		classifiedErr := s.errorClassifier.Classify(fmt.Errorf("failed to get KMS provider: %w", err), "GetKey")
		classifiedErr.KeyID = keyID.String()
		return nil, s.errorClassifier.LogAndSanitize(ctx, classifiedErr)
	}

	decryptedDEK, err := kmsProvider.DecryptDEK(ctx, key)
	if err != nil {
		classifiedErr := s.errorClassifier.Classify(fmt.Errorf("%w: %w", app_errors.ErrKMSFailure, err), "GetKey")
		classifiedErr.KeyID = keyID.String()
		return nil, s.errorClassifier.LogAndSanitize(ctx, classifiedErr)
	}
	defer secureZeroBytes(decryptedDEK)

	_, algorithm, err := getCryptoDetails(key.Metadata.GetKeyType())
	if err != nil {
		classifiedErr := s.errorClassifier.Classify(err, "GetKey")
		classifiedErr.KeyID = keyID.String()
		return nil, s.errorClassifier.LogAndSanitize(ctx, classifiedErr)
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
		return nil, status.Errorf(codes.InvalidArgument, "invalid key id: %v", err)
	}

	key, err := s.getKeyByRequest(ctx, keyID, req.GetVersion())
	if err != nil {
		if errors.Is(err, persistence.ErrKeyNotFound) {
			return nil, status.Errorf(codes.NotFound, "key not found: %s", req.GetKeyId())
		}
		s.logger.ErrorContext(ctx, "failed to get key metadata", "keyId", req.GetKeyId(), "error", err)
		return nil, status.Errorf(codes.Internal, "failed to retrieve key metadata")
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
