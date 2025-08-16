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

	kmsProvider, err := s.getKMSProvider(key.Metadata.GetDataClassification())
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

	key, err := s.getKeyByRequest(ctx, keyID, req.GetVersion())
	if err != nil {
		s.logger.ErrorContext(ctx, "[key_retriever.go:GetKeyMetadata] Error from getKeyByRequest", "error", err)
		return nil, err
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

	s.auditLogger.AuditLog(ctx, req.GetRequesterContext().GetClientIdentity(), "GetKeyMetadata", keyID.String(), "", true, nil)
	s.logger.InfoContext(ctx, "key metadata retrieved", "keyId", req.GetKeyId(), "version", key.Version)
	return resp, nil
}
