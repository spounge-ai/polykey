package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/service"
	"github.com/spounge-ai/polykey/internal/validation"
	"github.com/spounge-ai/polykey/pkg/errors"
	cmn "github.com/spounge-ai/spounge-proto/gen/go/common/v2"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// PolykeyService is the gRPC implementation of the pk.PolykeyServiceServer interface.
// It acts as the transport layer, delegating business logic to various services.
type PolykeyService struct {
	pk.UnimplementedPolykeyServiceServer
	cfg             *config.Config
	keyService      service.KeyService
	authService     service.AuthService
	authorizer      domain.Authorizer
	audit           domain.AuditLogger
	logger          *slog.Logger
	errorClassifier *errors.ErrorClassifier
	queryValidator  *validation.QueryValidator
}

// NewPolykeyService creates a new gRPC service implementation.
func NewPolykeyService(
	cfg *config.Config,
	keyService service.KeyService,
	authService service.AuthService,
	authorizer domain.Authorizer,
	audit domain.AuditLogger,
	logger *slog.Logger,
	errorClassifier *errors.ErrorClassifier,
	queryValidator *validation.QueryValidator,
) (pk.PolykeyServiceServer, error) {
	return &PolykeyService{
		cfg:             cfg,
		keyService:      keyService,
		authService:     authService,
		authorizer:      authorizer,
		audit:           audit,
		logger:          logger,
		errorClassifier: errorClassifier,
		queryValidator:  queryValidator,
	}, nil
}

// --- Authentication Methods ---

func (s *PolykeyService) Authenticate(ctx context.Context, req *pk.AuthenticateRequest) (*pk.AuthenticateResponse, error) {
	if req.GetClientId() == "" || req.GetApiKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "client_id and api_key are required")
	}

	result, err := s.authService.Authenticate(ctx, req.GetClientId(), req.GetApiKey())
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "authentication failed: %v", err)
	}

	return &pk.AuthenticateResponse{
		AccessToken: result.AccessToken,
		TokenType:   result.TokenType,
		ExpiresIn:   result.ExpiresIn,
		IssuedAt:    timestamppb.Now(),
		ClientTier:  cmn.ClientTier(cmn.ClientTier_value[strings.ToUpper(string(result.ClientTier))]),
	}, nil
}

// --- Key Management Methods ---

func (s *PolykeyService) GetKey(ctx context.Context, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error) {
	keyID, err := domain.KeyIDFromString(req.GetKeyId())
	if err != nil {
		classified := s.errorClassifier.Classify(errors.ErrInvalidInput, "GetKey")
		classified.Metadata["details"] = err.Error()
		classified.Metadata["requested_id"] = req.GetKeyId()
		return nil, s.errorClassifier.LogAndSanitize(ctx, classified)
	}

	if ok, reason := s.authorizer.Authorize(ctx, nil, nil, "keys:read", keyID); !ok {
		authErr := fmt.Errorf("authorization failed: %s", reason)
		classified := s.errorClassifier.Classify(errors.ErrAuthorization, "GetKey")
		classified.InternalError = authErr
		classified.KeyID = keyID.String()
		return nil, s.errorClassifier.LogAndSanitize(ctx, classified)
	}

	resp, err := s.keyService.GetKey(ctx, req)
	if err != nil {
		classified := s.errorClassifier.Classify(err, "GetKey")
		classified.KeyID = keyID.String()
		return nil, s.errorClassifier.LogAndSanitize(ctx, classified)
	}
	return resp, nil
}

func (s *PolykeyService) CreateKey(ctx context.Context, req *pk.CreateKeyRequest) (*pk.CreateKeyResponse, error) {
	if ok, reason := s.authorizer.Authorize(ctx, nil, nil, "keys:create", domain.KeyID{}); !ok {
		return nil, status.Errorf(codes.PermissionDenied, "authorization failed: %s", reason)
	}

	resp, err := s.keyService.CreateKey(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create key: %v", err)
	}
	return resp, nil
}

func (s *PolykeyService) ListKeys(ctx context.Context, req *pk.ListKeysRequest) (*pk.ListKeysResponse, error) {
	if err := s.queryValidator.ValidateListKeysRequest(req); err != nil {
		classified := s.errorClassifier.Classify(errors.ErrInvalidInput, "ListKeys")
		classified.Metadata["details"] = err.Error()
		return nil, s.errorClassifier.LogAndSanitize(ctx, classified)
	}

	if ok, reason := s.authorizer.Authorize(ctx, nil, nil, "keys:list", domain.KeyID{}); !ok {
		authErr := fmt.Errorf("authorization failed: %s", reason)
		classified := s.errorClassifier.Classify(errors.ErrAuthorization, "ListKeys")
		classified.InternalError = authErr
		return nil, s.errorClassifier.LogAndSanitize(ctx, classified)
	}

	resp, err := s.keyService.ListKeys(ctx, req)
	if err != nil {
		classified := s.errorClassifier.Classify(err, "ListKeys")
		return nil, s.errorClassifier.LogAndSanitize(ctx, classified)
	}
	return resp, nil
}

func (s *PolykeyService) RotateKey(ctx context.Context, req *pk.RotateKeyRequest) (*pk.RotateKeyResponse, error) {
	keyID, err := domain.KeyIDFromString(req.GetKeyId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid key id: %v", err)
	}

	if ok, reason := s.authorizer.Authorize(ctx, nil, nil, "keys:rotate", keyID); !ok {
		return nil, status.Errorf(codes.PermissionDenied, "authorization failed: %s", reason)
	}

	resp, err := s.keyService.RotateKey(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to rotate key: %v", err)
	}
	return resp, nil
}

func (s *PolykeyService) RevokeKey(ctx context.Context, req *pk.RevokeKeyRequest) (*emptypb.Empty, error) {
	keyID, err := domain.KeyIDFromString(req.GetKeyId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid key id: %v", err)
	}

	if ok, reason := s.authorizer.Authorize(ctx, nil, nil, "keys:revoke", keyID); !ok {
		return nil, status.Errorf(codes.PermissionDenied, "authorization failed: %s", reason)
	}

	if err := s.keyService.RevokeKey(ctx, req); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to revoke key: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (s *PolykeyService) UpdateKeyMetadata(ctx context.Context, req *pk.UpdateKeyMetadataRequest) (*emptypb.Empty, error) {
	keyID, err := domain.KeyIDFromString(req.GetKeyId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid key id: %v", err)
	}

	if ok, reason := s.authorizer.Authorize(ctx, nil, nil, "keys:update", keyID); !ok {
		return nil, status.Errorf(codes.PermissionDenied, "authorization failed: %s", reason)
	}

	if err := s.keyService.UpdateKeyMetadata(ctx, req); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update key metadata: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (s *PolykeyService) GetKeyMetadata(ctx context.Context, req *pk.GetKeyMetadataRequest) (*pk.GetKeyMetadataResponse, error) {
	keyID, err := domain.KeyIDFromString(req.GetKeyId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid key id: %v", err)
	}

	if ok, reason := s.authorizer.Authorize(ctx, nil, nil, "keys:read", keyID); !ok {
		return nil, status.Errorf(codes.PermissionDenied, "authorization failed: %s", reason)
	}

	resp, err := s.keyService.GetKeyMetadata(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get key metadata: %v", err)
	}
	return resp, nil
}

func (s *PolykeyService) HealthCheck(ctx context.Context, req *emptypb.Empty) (*pk.HealthCheckResponse, error) {
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