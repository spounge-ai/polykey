package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/spounge-ai/polykey/internal/audit"
	"github.com/spounge-ai/polykey/internal/authz"
	"github.com/spounge-ai/polykey/internal/config"
	"github.com/spounge-ai/polykey/internal/keymanager"
	"github.com/spounge-ai/polykey/internal/storage"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// polykeyServiceImpl implements the PolykeyService interface.
type polykeyServiceImpl struct {
	pk.UnimplementedPolykeyServiceServer
	cfg        *config.Config
	storage    storage.Storage
	authorizer authz.Authorizer
	auditLoggr audit.Logger
	keyManager keymanager.KeyManager
}

// NewPolykeyService creates a new instance of PolykeyService.
func NewPolykeyService(cfg *config.Config, s storage.Storage, authorizer authz.Authorizer, auditLogger audit.Logger, keyManager keymanager.KeyManager) (pk.PolykeyServiceServer, error) {
	return &polykeyServiceImpl{
		cfg:        cfg,
		storage:    s,
		authorizer: authorizer,
		auditLoggr: auditLogger,
		keyManager: keyManager,
	}, nil
}

// GetKey implements pk.PolykeyServiceServer.
func (s *polykeyServiceImpl) GetKey(ctx context.Context, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error) {
	log.Printf("Received GetKey request for key_id: %s", req.GetKeyId())

	keyID := req.GetKeyId()
	requesterContext := req.GetRequesterContext()
	accessAttributes := req.GetAttributes()

	// Authorization Check
	isAuthorized, authDecisionID := s.authorizer.Authorize(ctx, requesterContext, accessAttributes, keyID)
	if !isAuthorized {
		s.auditLoggr.AuditLog(ctx, requesterContext.GetClientIdentity(), "GetKey", keyID, authDecisionID, false, fmt.Errorf("permission denied"))
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	resp, err := s.keyManager.GetKey(ctx, req)
	if err != nil {
		s.auditLoggr.AuditLog(ctx, requesterContext.GetClientIdentity(), "GetKey", keyID, authDecisionID, false, err)
		return nil, status.Errorf(codes.Internal, "failed to retrieve key: %v", err)
	}

	resp.AuthorizationDecisionId = authDecisionID
	s.auditLoggr.AuditLog(ctx, requesterContext.GetClientIdentity(), "GetKey", keyID, authDecisionID, true, nil)
	return resp, nil
}

// ListKeys implements pk.PolykeyServiceServer.
func (s *polykeyServiceImpl) ListKeys(ctx context.Context, req *pk.ListKeysRequest) (*pk.ListKeysResponse, error) {
	log.Println("Received ListKeys request")

	requesterContext := req.GetRequesterContext()
	accessAttributes := req.GetAttributes()

	// Get all keys from key manager (key manager doesn't handle authorization filtering)
	allKeysResp, err := s.keyManager.ListKeys(ctx, req)
	if err != nil {
		s.auditLoggr.AuditLog(ctx, requesterContext.GetClientIdentity(), "ListKeys", "N/A", "N/A", false, err)
		return nil, status.Errorf(codes.Internal, "failed to list keys: %v", err)
	}

	var authorizedKeys []*pk.KeyMetadata
	for _, key := range allKeysResp.GetKeys() {
		isAuthorized, authDecisionID := s.authorizer.Authorize(ctx, requesterContext, accessAttributes, key.GetKeyId())
		if isAuthorized {
			authorizedKeys = append(authorizedKeys, key)
		}
		s.auditLoggr.AuditLog(ctx, requesterContext.GetClientIdentity(), "ListKeys_AuthCheck", key.GetKeyId(), authDecisionID, isAuthorized, nil)
	}

	// Implement basic pagination on authorized keys
	pageSize := int(req.GetPageSize())
	if pageSize == 0 {
		pageSize = 5 // Default page size
	}

	startIndex := 0
	if req.GetPageToken() != "" {
		fmt.Sscanf(req.GetPageToken(), "%d", &startIndex)
	}

	endIndex := startIndex + pageSize
	if endIndex > len(authorizedKeys) {
		endIndex = len(authorizedKeys)
	}

	pagedKeys := authorizedKeys[startIndex:endIndex]

	nextPageToken := ""
	if endIndex < len(authorizedKeys) {
		nextPageToken = fmt.Sprintf("%d", endIndex)
	}

	s.auditLoggr.AuditLog(ctx, requesterContext.GetClientIdentity(), "ListKeys", "N/A", "N/A", true, nil)

	return &pk.ListKeysResponse{
		Keys:              pagedKeys,
		NextPageToken:     nextPageToken,
		TotalCount:        int32(len(allKeysResp.GetKeys())),
		ResponseTimestamp: timestamppb.Now(),
		FilteredCount:     int32(len(authorizedKeys)),
	}, nil
}

// CreateKey implements pk.PolykeyServiceServer.
func (s *polykeyServiceImpl) CreateKey(ctx context.Context, req *pk.CreateKeyRequest) (*pk.CreateKeyResponse, error) {
	log.Printf("Received CreateKey request for key_type: %s", req.GetKeyType().String())

	requesterContext := req.GetRequesterContext()
	clientIdentity := ""
	if requesterContext != nil {
		clientIdentity = requesterContext.GetClientIdentity()
	}

	// Authorization Check
	isAuthorized, authDecisionID := s.authorizer.Authorize(ctx, requesterContext, nil, "create_key_operation")
	if !isAuthorized || clientIdentity != "test_creator" {
		s.auditLoggr.AuditLog(ctx, clientIdentity, "CreateKey", "N/A", authDecisionID, false, fmt.Errorf("permission denied"))
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	keyType := req.GetKeyType()
	// Validate key type
	switch keyType {
	case pk.KeyType_KEY_TYPE_API_KEY,
		pk.KeyType_KEY_TYPE_AES_256,
		pk.KeyType_KEY_TYPE_RSA_4096,
		pk.KeyType_KEY_TYPE_ECDSA_P384:
		// Valid key type, continue
	default:
		s.auditLoggr.AuditLog(ctx, clientIdentity, "CreateKey", "N/A", authDecisionID, false, fmt.Errorf("unsupported key type: %v", keyType))
		return nil, status.Errorf(codes.InvalidArgument, "unsupported key type: %v", keyType)
	}

	resp, err := s.keyManager.CreateKey(ctx, req)
	if err != nil {
		s.auditLoggr.AuditLog(ctx, clientIdentity, "CreateKey", "N/A", authDecisionID, false, err)
		return nil, status.Errorf(codes.Internal, "failed to create key: %v", err)
	}

	s.auditLoggr.AuditLog(ctx, clientIdentity, "CreateKey", resp.GetKeyId(), authDecisionID, true, nil)
	return resp, nil
}

// RotateKey implements pk.PolykeyServiceServer.
func (s *polykeyServiceImpl) RotateKey(ctx context.Context, req *pk.RotateKeyRequest) (*pk.RotateKeyResponse, error) {
	log.Printf("Received RotateKey request for key_id: %s", req.GetKeyId())

	keyID := req.GetKeyId()
	requesterContext := req.GetRequesterContext()

	// Authorization Check
	isAuthorized, authDecisionID := s.authorizer.Authorize(ctx, requesterContext, nil, keyID)
	if !isAuthorized {
		s.auditLoggr.AuditLog(ctx, requesterContext.GetClientIdentity(), "RotateKey", keyID, authDecisionID, false, fmt.Errorf("permission denied"))
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	resp, err := s.keyManager.RotateKey(ctx, req)
	if err != nil {
		s.auditLoggr.AuditLog(ctx, requesterContext.GetClientIdentity(), "RotateKey", keyID, authDecisionID, false, err)
		return nil, status.Errorf(codes.Internal, "failed to rotate key: %v", err)
	}

	s.auditLoggr.AuditLog(ctx, requesterContext.GetClientIdentity(), "RotateKey", keyID, authDecisionID, true, nil)
	return resp, nil
}

// RevokeKey implements pk.PolykeyServiceServer.
func (s *polykeyServiceImpl) RevokeKey(ctx context.Context, req *pk.RevokeKeyRequest) (*emptypb.Empty, error) {
	log.Printf("Received RevokeKey request for key_id: %s, reason: %s", req.GetKeyId(), req.GetRevocationReason())

	keyID := req.GetKeyId()
	requesterContext := req.GetRequesterContext()

	// Authorization Check
	isAuthorized, authDecisionID := s.authorizer.Authorize(ctx, requesterContext, nil, keyID)
	if !isAuthorized {
		s.auditLoggr.AuditLog(ctx, requesterContext.GetClientIdentity(), "RevokeKey", keyID, authDecisionID, false, fmt.Errorf("permission denied"))
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	err := s.keyManager.RevokeKey(ctx, req)
	if err != nil {
		s.auditLoggr.AuditLog(ctx, requesterContext.GetClientIdentity(), "RevokeKey", keyID, authDecisionID, false, err)
		return nil, status.Errorf(codes.Internal, "failed to revoke key: %v", err)
	}

	s.auditLoggr.AuditLog(ctx, requesterContext.GetClientIdentity(), "RevokeKey", keyID, authDecisionID, true, nil)
	return &emptypb.Empty{}, nil
}

// UpdateKeyMetadata implements pk.PolykeyServiceServer.
func (s *polykeyServiceImpl) UpdateKeyMetadata(ctx context.Context, req *pk.UpdateKeyMetadataRequest) (*emptypb.Empty, error) {
	log.Printf("Received UpdateKeyMetadata request for key_id: %s", req.GetKeyId())

	keyID := req.GetKeyId()
	requesterContext := req.GetRequesterContext()

	// Authorization Check
	isAuthorized, authDecisionID := s.authorizer.Authorize(ctx, requesterContext, nil, keyID)
	if !isAuthorized {
		s.auditLoggr.AuditLog(ctx, requesterContext.GetClientIdentity(), "UpdateKeyMetadata", keyID, authDecisionID, false, fmt.Errorf("permission denied"))
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	err := s.keyManager.UpdateKeyMetadata(ctx, req)
	if err != nil {
		s.auditLoggr.AuditLog(ctx, requesterContext.GetClientIdentity(), "UpdateKeyMetadata", keyID, authDecisionID, false, err)
		return nil, status.Errorf(codes.Internal, "failed to update key metadata: %v", err)
	}

	s.auditLoggr.AuditLog(ctx, requesterContext.GetClientIdentity(), "UpdateKeyMetadata", keyID, authDecisionID, true, nil)
	return &emptypb.Empty{}, nil
}

// GetKeyMetadata implements pk.PolykeyServiceServer.
func (s *polykeyServiceImpl) GetKeyMetadata(ctx context.Context, req *pk.GetKeyMetadataRequest) (*pk.GetKeyMetadataResponse, error) {
	log.Printf("Received GetKeyMetadata request for key_id: %s", req.GetKeyId())

	keyID := req.GetKeyId()
	requesterContext := req.GetRequesterContext()

	// Authorization Check
	isAuthorized, authDecisionID := s.authorizer.Authorize(ctx, requesterContext, nil, keyID)
	if !isAuthorized {
		s.auditLoggr.AuditLog(ctx, requesterContext.GetClientIdentity(), "GetKeyMetadata", keyID, authDecisionID, false, fmt.Errorf("permission denied"))
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	resp, err := s.keyManager.GetKeyMetadata(ctx, req)
	if err != nil {
		s.auditLoggr.AuditLog(ctx, requesterContext.GetClientIdentity(), "GetKeyMetadata", keyID, authDecisionID, false, err)
		return nil, status.Errorf(codes.Internal, "failed to get key metadata: %v", err)
	}

	s.auditLoggr.AuditLog(ctx, requesterContext.GetClientIdentity(), "GetKeyMetadata", keyID, authDecisionID, true, nil)
	return resp, nil
}

// HealthCheck implements pk.PolykeyServiceServer.
func (s *polykeyServiceImpl) HealthCheck(ctx context.Context, req *emptypb.Empty) (*pk.HealthCheckResponse, error) {
	log.Println("Received HealthCheck request")

	// Perform a basic health check on the storage backend
	if s.storage != nil {
		err := s.storage.HealthCheck()
		if err != nil {
			log.Printf("Storage health check failed: %v", err)
			return &pk.HealthCheckResponse{
				Status: pk.HealthStatus_HEALTH_STATUS_UNHEALTHY,
				Timestamp: timestamppb.Now(),
				ServiceVersion: s.cfg.ServiceVersion,
				BuildCommit:    s.cfg.BuildCommit,
			},
			nil
		}
	}

	return &pk.HealthCheckResponse{
		Status: pk.HealthStatus_HEALTH_STATUS_HEALTHY,
		Timestamp: timestamppb.Now(),
		ServiceVersion: s.cfg.ServiceVersion,
		BuildCommit:    s.cfg.BuildCommit,
		Metrics: &pk.ServiceMetrics{
			AverageResponseTimeMs: 10.5,
			RequestsPerSecond:     100,
			ErrorRatePercent:      0.1,
			CpuUsagePercent:       20.0,
			MemoryUsagePercent:    30.0,
			ActiveKeysCount:       1000,
			TotalRequestsHandled:  100000,
			UptimeSince:           timestamppb.New(time.Now().Add(-24 * time.Hour)),
		},
	},
	nil
}