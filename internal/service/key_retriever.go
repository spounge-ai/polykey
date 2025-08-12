package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/internal/infra/persistence" // Add this import
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *keyServiceImpl) GetKey(ctx context.Context, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: request is nil", ErrInvalidRequest)
	}
	keyID, err := domain.KeyIDFromString(req.GetKeyId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid key id: %v", err)
	}

	key, err := s.getKeyByRequest(ctx, keyID, req.GetVersion())
	if err != nil {
		// Check if the error contains a gRPC status error (even if wrapped)
		var currentErr error = err
		for currentErr != nil {
			if statusErr, ok := status.FromError(currentErr); ok {
				// Found a gRPC status error, return it
				return nil, statusErr.Err()
			}
			// Check if it's a wrapped error and unwrap it
			if wrappedErr := errors.Unwrap(currentErr); wrappedErr != nil {
				currentErr = wrappedErr
			} else {
				break
			}
		}
		
		// Convert storage errors to appropriate gRPC status codes
		if errors.Is(err, persistence.ErrKeyNotFound) {
			return nil, status.Errorf(codes.NotFound, "key not found: %s", req.GetKeyId())
		}
		
		s.logger.ErrorContext(ctx, "failed to get key from repository", "keyId", req.GetKeyId(), "error", err)
		return nil, status.Errorf(codes.Internal, "failed to retrieve key")
	}

	if key.Metadata == nil {
		return nil, status.Errorf(codes.Internal, "key metadata is missing")
	}

	kmsProvider, err := s.getKMSProvider(key.Metadata.GetDataClassification())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get KMS provider: %v", err)
	}

	decryptedDEK, err := kmsProvider.DecryptDEK(ctx, key)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to decrypt DEK", "keyId", req.GetKeyId(), "error", err)
		return nil, status.Errorf(codes.Internal, "failed to decrypt key material")
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
		return nil, status.Errorf(codes.InvalidArgument, "invalid key id: %v", err)
	}

	key, err := s.getKeyByRequest(ctx, keyID, req.GetVersion())
	if err != nil {
		// Convert storage errors to appropriate gRPC status codes
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