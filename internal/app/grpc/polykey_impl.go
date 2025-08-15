package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/spounge-ai/polykey/internal/domain"
	app_errors "github.com/spounge-ai/polykey/internal/errors"
	"github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/service"
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
	errorClassifier *app_errors.ErrorClassifier
}

// NewPolykeyService creates a new gRPC service implementation.
func NewPolykeyService(
	cfg *config.Config,
	keyService service.KeyService,
	authService service.AuthService,
	authorizer domain.Authorizer,
	audit domain.AuditLogger,
	logger *slog.Logger,
	errorClassifier *app_errors.ErrorClassifier,
) (pk.PolykeyServiceServer, error) {
	return &PolykeyService{
		cfg:             cfg,
		keyService:      keyService,
		authService:     authService,
		authorizer:      authorizer,
		audit:           audit,
		logger:          logger,
		errorClassifier: errorClassifier,
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
	if _, err := s.authorizeWithKeyID(ctx, "GetKey", "keys:read", req.GetKeyId()); err != nil {
		return nil, err
	}

	resp, err := s.keyService.GetKey(ctx, req)
	if err != nil {
		return nil, s.errorClassifier.LogAndSanitize(ctx, s.errorClassifier.Classify(err, "GetKey"))
	}
	return resp, nil
}

func (s *PolykeyService) CreateKey(ctx context.Context, req *pk.CreateKeyRequest) (*pk.CreateKeyResponse, error) {
	if err := s.authorize(ctx, "CreateKey", "keys:create"); err != nil {
		return nil, err
	}

	resp, err := s.keyService.CreateKey(ctx, req)
	if err != nil {
		return nil, s.errorClassifier.LogAndSanitize(ctx, s.errorClassifier.Classify(err, "CreateKey"))
	}
	return resp, nil
}

func (s *PolykeyService) ListKeys(ctx context.Context, req *pk.ListKeysRequest) (*pk.ListKeysResponse, error) {
	if err := s.authorize(ctx, "ListKeys", "keys:list"); err != nil {
		return nil, err
	}

	resp, err := s.keyService.ListKeys(ctx, req)
	if err != nil {
		return nil, s.errorClassifier.LogAndSanitize(ctx, s.errorClassifier.Classify(err, "ListKeys"))
	}
	return resp, nil
}

func (s *PolykeyService) RotateKey(ctx context.Context, req *pk.RotateKeyRequest) (*pk.RotateKeyResponse, error) {
	if _, err := s.authorizeWithKeyID(ctx, "RotateKey", "keys:rotate", req.GetKeyId()); err != nil {
		return nil, err
	}

	resp, err := s.keyService.RotateKey(ctx, req)
	if err != nil {
		return nil, s.errorClassifier.LogAndSanitize(ctx, s.errorClassifier.Classify(err, "RotateKey"))
	}
	return resp, nil
}

func (s *PolykeyService) RevokeKey(ctx context.Context, req *pk.RevokeKeyRequest) (*emptypb.Empty, error) {
	if _, err := s.authorizeWithKeyID(ctx, "RevokeKey", "keys:revoke", req.GetKeyId()); err != nil {
		return nil, err
	}

	if err := s.keyService.RevokeKey(ctx, req); err != nil {
		return nil, s.errorClassifier.LogAndSanitize(ctx, s.errorClassifier.Classify(err, "RevokeKey"))
	}
	return &emptypb.Empty{}, nil
}

func (s *PolykeyService) UpdateKeyMetadata(ctx context.Context, req *pk.UpdateKeyMetadataRequest) (*emptypb.Empty, error) {
	if _, err := s.authorizeWithKeyID(ctx, "UpdateKeyMetadata", "keys:update", req.GetKeyId()); err != nil {
		return nil, err
	}

	if err := s.keyService.UpdateKeyMetadata(ctx, req); err != nil {
		return nil, s.errorClassifier.LogAndSanitize(ctx, s.errorClassifier.Classify(err, "UpdateKeyMetadata"))
	}
	return &emptypb.Empty{}, nil
}

func (s *PolykeyService) GetKeyMetadata(ctx context.Context, req *pk.GetKeyMetadataRequest) (*pk.GetKeyMetadataResponse, error) {
	if _, err := s.authorizeWithKeyID(ctx, "GetKeyMetadata", "keys:read", req.GetKeyId()); err != nil {
		return nil, err
	}

	resp, err := s.keyService.GetKeyMetadata(ctx, req)
	if err != nil {
		return nil, s.errorClassifier.LogAndSanitize(ctx, s.errorClassifier.Classify(err, "GetKeyMetadata"))
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

// --- Helper Functions ---

func (s *PolykeyService) authorize(ctx context.Context, methodName, authOperation string) error {
	if ok, reason := s.authorizer.Authorize(ctx, nil, nil, authOperation, domain.KeyID{}); !ok {
		return s.errorClassifier.LogAndSanitize(ctx, s.errorClassifier.Classify(fmt.Errorf("%w: %s", app_errors.ErrAuthorization, reason), methodName))
	}
	return nil
}

func (s *PolykeyService) authorizeWithKeyID(ctx context.Context, methodName, authOperation, keyIdStr string) (domain.KeyID, error) {
	keyID, err := domain.KeyIDFromString(keyIdStr)
	if err != nil {
		return domain.KeyID{}, s.errorClassifier.LogAndSanitize(ctx, s.errorClassifier.Classify(err, methodName))
	}

	if ok, reason := s.authorizer.Authorize(ctx, nil, nil, authOperation, keyID); !ok {
		return domain.KeyID{}, s.errorClassifier.LogAndSanitize(ctx, s.errorClassifier.Classify(fmt.Errorf("%w: %s", app_errors.ErrAuthorization, reason), methodName))
	}
	return keyID, nil
}