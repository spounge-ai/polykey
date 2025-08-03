package grpc

import (
	"context"
	"crypto/rand"
	"time"

	"github.com/google/uuid"
	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/internal/infra/config"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// polykeyServiceImpl implements the PolykeyService interface.
type polykeyServiceImpl struct {
	pk.UnimplementedPolykeyServiceServer
	cfg        *config.Config
	keyRepo    domain.KeyRepository
	kms        domain.KMSService
	authorizer domain.Authorizer
	audit      domain.AuditLogger
}

// NewPolykeyService creates a new instance of PolykeyService.
func NewPolykeyService(cfg *config.Config, keyRepo domain.KeyRepository, kms domain.KMSService, authorizer domain.Authorizer, audit domain.AuditLogger) (pk.PolykeyServiceServer, error) {
	return &polykeyServiceImpl{
		cfg:        cfg,
		keyRepo:    keyRepo,
		kms:        kms,
		authorizer: authorizer,
		audit:      audit,
	}, nil
}

func (s *polykeyServiceImpl) GetKey(ctx context.Context, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error) {
	var key *domain.Key
	var err error

	if req.GetVersion() > 0 {
		key, err = s.keyRepo.GetKeyByVersion(ctx, req.GetKeyId(), req.GetVersion())
	} else {
		key, err = s.keyRepo.GetKey(ctx, req.GetKeyId())
	}

	if err != nil {
		s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "GetKey", req.GetKeyId(), "", false, err)
		return nil, status.Errorf(codes.NotFound, "key not found: %v", err)
	}

	// Authorize access
	authorized, authDecisionID := s.authorizer.Authorize(ctx, req.RequesterContext, req.Attributes, "GetKey")
	if !authorized {
		s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "GetKey", req.GetKeyId(), authDecisionID, false, nil)
		return nil, status.Errorf(codes.PermissionDenied, "access denied")
	}

	dek, err := s.kms.DecryptDEK(ctx, key.EncryptedDEK, "alias/polykey")
	if err != nil {
		s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "GetKey", req.GetKeyId(), authDecisionID, false, err)
		return nil, status.Errorf(codes.Internal, "failed to decrypt key: %v", err)
	}

	resp := &pk.GetKeyResponse{
		KeyMaterial: &pk.KeyMaterial{
			EncryptedKeyData:     dek,
			EncryptionAlgorithm:  "AES-256-GCM",
			KeyChecksum:          "sha256",
		},
		Metadata:                 key.Metadata,
		ResponseTimestamp:        timestamppb.Now(),
		AuthorizationDecisionId:  authDecisionID,
	}

	if !req.GetSkipMetadata() {
		resp.Metadata = key.Metadata
	}

	s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "GetKey", req.GetKeyId(), authDecisionID, true, nil)
	return resp, nil
}

func (s *polykeyServiceImpl) CreateKey(ctx context.Context, req *pk.CreateKeyRequest) (*pk.CreateKeyResponse, error) {
	// Authorize creation
	authorized, authDecisionID := s.authorizer.Authorize(ctx, req.RequesterContext, nil, "CreateKey")
	if !authorized {
		s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "CreateKey", "", authDecisionID, false, nil)
		return nil, status.Errorf(codes.PermissionDenied, "access denied")
	}

	// Generate DEK
	dek := make([]byte, 32)
	if _, err := rand.Read(dek); err != nil {
		s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "CreateKey", "", authDecisionID, false, err)
		return nil, status.Errorf(codes.Internal, "failed to generate DEK: %v", err)
	}

	encryptedDEK, err := s.kms.EncryptDEK(ctx, dek, "alias/polykey")
	if err != nil {
		s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "CreateKey", "", authDecisionID, false, err)
		return nil, status.Errorf(codes.Internal, "failed to encrypt DEK: %v", err)
	}

	keyID := uuid.New().String()
	now := time.Now()

	metadata := &pk.KeyMetadata{
		KeyId:              keyID,
		KeyType:            req.GetKeyType(),
		Status:             pk.KeyStatus_KEY_STATUS_ACTIVE,
		Version:            1,
		CreatedAt:          timestamppb.New(now),
		UpdatedAt:          timestamppb.New(now),
		ExpiresAt:          req.GetExpiresAt(),
		CreatorIdentity:    req.RequesterContext.GetClientIdentity(),
		AuthorizedContexts: req.GetInitialAuthorizedContexts(),
		AccessPolicies:     req.GetAccessPolicies(),
		Description:        req.GetDescription(),
		Tags:               req.GetTags(),
		DataClassification: req.GetDataClassification(),
		AccessCount:        0,
	}

	newKey := &domain.Key{
		ID:           keyID,
		Version:      1,
		Metadata:     metadata,
		EncryptedDEK: encryptedDEK,
		Status:       domain.KeyStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.keyRepo.CreateKey(ctx, newKey); err != nil {
		s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "CreateKey", keyID, authDecisionID, false, err)
		return nil, status.Errorf(codes.Internal, "failed to create key: %v", err)
	}

	resp := &pk.CreateKeyResponse{
		KeyId: keyID,
		Metadata: metadata,
		KeyMaterial: &pk.KeyMaterial{
			EncryptedKeyData:    dek,
			EncryptionAlgorithm: "AES-256-GCM",
			KeyChecksum:         "sha256",
		},
		ResponseTimestamp: timestamppb.Now(),
	}

	s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "CreateKey", keyID, authDecisionID, true, nil)
	return resp, nil
}

func (s *polykeyServiceImpl) ListKeys(ctx context.Context, req *pk.ListKeysRequest) (*pk.ListKeysResponse, error) {
	// Authorize listing
	authorized, authDecisionID := s.authorizer.Authorize(ctx, req.RequesterContext, req.Attributes, "ListKeys")
	if !authorized {
		s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "ListKeys", "", authDecisionID, false, nil)
		return nil, status.Errorf(codes.PermissionDenied, "access denied")
	}

	keys, err := s.keyRepo.ListKeys(ctx)
	if err != nil {
		s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "ListKeys", "", authDecisionID, false, err)
		return nil, status.Errorf(codes.Internal, "failed to list keys: %v", err)
	}

	var metadataKeys []*pk.KeyMetadata
	for _, key := range keys {
		metadataKeys = append(metadataKeys, key.Metadata)
	}

	resp := &pk.ListKeysResponse{
		Keys:              metadataKeys,
		TotalCount:        int32(len(metadataKeys)),
		FilteredCount:     int32(len(metadataKeys)),
		ResponseTimestamp: timestamppb.Now(),
	}

	s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "ListKeys", "", authDecisionID, true, nil)
	return resp, nil
}

func (s *polykeyServiceImpl) RotateKey(ctx context.Context, req *pk.RotateKeyRequest) (*pk.RotateKeyResponse, error) {
	// Authorize rotation
	authorized, authDecisionID := s.authorizer.Authorize(ctx, req.RequesterContext, nil, "RotateKey")
	if !authorized {
		s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "RotateKey", req.GetKeyId(), authDecisionID, false, nil)
		return nil, status.Errorf(codes.PermissionDenied, "access denied")
	}

	// Get current key
	currentKey, err := s.keyRepo.GetKey(ctx, req.GetKeyId())
	if err != nil {
		s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "RotateKey", req.GetKeyId(), authDecisionID, false, err)
		return nil, status.Errorf(codes.NotFound, "key not found: %v", err)
	}

	// Generate new DEK
	newDEK := make([]byte, 32)
	if _, err := rand.Read(newDEK); err != nil {
		s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "RotateKey", req.GetKeyId(), authDecisionID, false, err)
		return nil, status.Errorf(codes.Internal, "failed to generate new DEK: %v", err)
	}

	encryptedNewDEK, err := s.kms.EncryptDEK(ctx, newDEK, "alias/polykey")
	if err != nil {
		s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "RotateKey", req.GetKeyId(), authDecisionID, false, err)
		return nil, status.Errorf(codes.Internal, "failed to encrypt new DEK: %v", err)
	}

	// Rotate key in repository
	rotatedKey, err := s.keyRepo.RotateKey(ctx, req.GetKeyId(), encryptedNewDEK)
	if err != nil {
		s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "RotateKey", req.GetKeyId(), authDecisionID, false, err)
		return nil, status.Errorf(codes.Internal, "failed to rotate key: %v", err)
	}

	resp := &pk.RotateKeyResponse{
		KeyId:           req.GetKeyId(),
		NewVersion:      rotatedKey.Version,
		PreviousVersion: currentKey.Version,
		NewKeyMaterial: &pk.KeyMaterial{
			EncryptedKeyData:    newDEK,
			EncryptionAlgorithm: "AES-256-GCM",
			KeyChecksum:         "sha256",
		},
		Metadata:          rotatedKey.Metadata,
		RotationTimestamp: timestamppb.Now(),
		OldVersionExpiresAt: timestamppb.New(time.Now().Add(time.Duration(req.GetGracePeriodSeconds()) * time.Second)),
	}

	s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "RotateKey", req.GetKeyId(), authDecisionID, true, nil)
	return resp, nil
}

func (s *polykeyServiceImpl) RevokeKey(ctx context.Context, req *pk.RevokeKeyRequest) (*emptypb.Empty, error) {
	// Authorize revocation
	authorized, authDecisionID := s.authorizer.Authorize(ctx, req.RequesterContext, nil, "RevokeKey")
	if !authorized {
		s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "RevokeKey", req.GetKeyId(), authDecisionID, false, nil)
		return nil, status.Errorf(codes.PermissionDenied, "access denied")
	}

	if err := s.keyRepo.RevokeKey(ctx, req.GetKeyId()); err != nil {
		s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "RevokeKey", req.GetKeyId(), authDecisionID, false, err)
		return nil, status.Errorf(codes.Internal, "failed to revoke key: %v", err)
	}

	s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "RevokeKey", req.GetKeyId(), authDecisionID, true, nil)
	return &emptypb.Empty{}, nil
}

func (s *polykeyServiceImpl) UpdateKeyMetadata(ctx context.Context, req *pk.UpdateKeyMetadataRequest) (*emptypb.Empty, error) {
	// Authorize update
	authorized, authDecisionID := s.authorizer.Authorize(ctx, req.RequesterContext, nil, "UpdateKeyMetadata")
	if !authorized {
		s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "UpdateKeyMetadata", req.GetKeyId(), authDecisionID, false, nil)
		return nil, status.Errorf(codes.PermissionDenied, "access denied")
	}

	// Get current metadata
	key, err := s.keyRepo.GetKey(ctx, req.GetKeyId())
	if err != nil {
		s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "UpdateKeyMetadata", req.GetKeyId(), authDecisionID, false, err)
		return nil, status.Errorf(codes.NotFound, "key not found: %v", err)
	}

	// Update metadata fields
	metadata := key.Metadata
	if req.Description != nil {
		metadata.Description = *req.Description
	}
	if req.ExpiresAt != nil {
		metadata.ExpiresAt = req.ExpiresAt
	}
	if req.DataClassification != nil {
		metadata.DataClassification = *req.DataClassification
	}

	// Update tags
	if metadata.Tags == nil {
		metadata.Tags = make(map[string]string)
	}
	for k, v := range req.GetTagsToAdd() {
		metadata.Tags[k] = v
	}
	for _, tag := range req.GetTagsToRemove() {
		delete(metadata.Tags, tag)
	}

	metadata.UpdatedAt = timestamppb.Now()

	if err := s.keyRepo.UpdateKeyMetadata(ctx, req.GetKeyId(), metadata); err != nil {
		s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "UpdateKeyMetadata", req.GetKeyId(), authDecisionID, false, err)
		return nil, status.Errorf(codes.Internal, "failed to update key metadata: %v", err)
	}

	s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "UpdateKeyMetadata", req.GetKeyId(), authDecisionID, true, nil)
	return &emptypb.Empty{}, nil
}

func (s *polykeyServiceImpl) GetKeyMetadata(ctx context.Context, req *pk.GetKeyMetadataRequest) (*pk.GetKeyMetadataResponse, error) {
	// Authorize access
	authorized, authDecisionID := s.authorizer.Authorize(ctx, req.RequesterContext, req.Attributes, "GetKeyMetadata")
	if !authorized {
		s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "GetKeyMetadata", req.GetKeyId(), authDecisionID, false, nil)
		return nil, status.Errorf(codes.PermissionDenied, "access denied")
	}

	var key *domain.Key
	var err error

	if req.GetVersion() > 0 {
		key, err = s.keyRepo.GetKeyByVersion(ctx, req.GetKeyId(), req.GetVersion())
	} else {
		key, err = s.keyRepo.GetKey(ctx, req.GetKeyId())
	}

	if err != nil {
		s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "GetKeyMetadata", req.GetKeyId(), authDecisionID, false, err)
		return nil, status.Errorf(codes.NotFound, "key not found: %v", err)
	}

	resp := &pk.GetKeyMetadataResponse{
		Metadata:          key.Metadata,
		ResponseTimestamp: timestamppb.Now(),
	}

	// TODO: Add access history if requested
	if req.GetIncludeAccessHistory() {
		// Implement audit history retrieval
	}

	// TODO: Add policy details if requested
	if req.GetIncludePolicyDetails() {
		// Implement policy details retrieval
	}

	s.audit.AuditLog(ctx, req.RequesterContext.GetClientIdentity(), "GetKeyMetadata", req.GetKeyId(), authDecisionID, true, nil)
	return resp, nil
}

func (s *polykeyServiceImpl) HealthCheck(ctx context.Context, req *emptypb.Empty) (*pk.HealthCheckResponse, error) {
	return &pk.HealthCheckResponse{
		Status:         pk.HealthStatus_HEALTH_STATUS_HEALTHY,
		Timestamp:      timestamppb.Now(),
		ServiceVersion: s.cfg.ServiceVersion,
		BuildCommit:    s.cfg.BuildCommit,
		Metrics: &pk.ServiceMetrics{
			UptimeSince: timestamppb.New(time.Now().Add(-24 * time.Hour)), // Mock uptime
		},
	}, nil
}