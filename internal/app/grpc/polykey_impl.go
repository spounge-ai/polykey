package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	cts "github.com/spounge-ai/polykey/internal/constants"
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

// PolykeyService implements pk.PolykeyServiceServer.
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

// --- Generic Helpers ---

func execWithAuth[T any](
	s *PolykeyService,
	ctx context.Context,
	methodName, authOp, keyIDStr string,
	fn func(context.Context, domain.KeyID) (T, error),
) (T, error) {
	var zero T

	keyID, err := domain.KeyIDFromString(keyIDStr)
	if err != nil {
		return zero, s.sanitizeError(ctx, methodName, err)
	}

	if ok, reason := s.authorizer.Authorize(ctx, nil, nil, authOp, keyID); !ok {
		return zero, s.sanitizeError(ctx, methodName, fmt.Errorf("%w: %s", app_errors.ErrAuthorization, reason))
	}

	res, err := fn(ctx, keyID)
	if err != nil {
		return zero, s.sanitizeError(ctx, methodName, err)
	}
	return res, nil
}

func execWithoutKey[T any](
	s *PolykeyService,
	ctx context.Context,
	methodName, authOp string,
	fn func(context.Context) (T, error),
) (T, error) {
	var zero T

	if ok, reason := s.authorizer.Authorize(ctx, nil, nil, authOp, domain.KeyID{}); !ok {
		return zero, s.sanitizeError(ctx, methodName, fmt.Errorf("%w: %s", app_errors.ErrAuthorization, reason))
	}

	res, err := fn(ctx)
	if err != nil {
		return zero, s.sanitizeError(ctx, methodName, err)
	}
	return res, nil
}

func (s *PolykeyService) sanitizeError(ctx context.Context, method string, err error) error {
	return s.errorClassifier.LogAndSanitize(ctx, s.errorClassifier.Classify(err, method))
}

// --- gRPC Methods ---

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

func (s *PolykeyService) GetKey(ctx context.Context, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error) {
	return execWithAuth(s, ctx,
		cts.MethodGetKey,
		cts.MethodScopes[cts.MethodGetKey],
		req.GetKeyId(),
		func(ctx context.Context, keyID domain.KeyID) (*pk.GetKeyResponse, error) {
			return s.keyService.GetKey(ctx, req)
		},
	)
}

func (s *PolykeyService) CreateKey(ctx context.Context, req *pk.CreateKeyRequest) (*pk.CreateKeyResponse, error) {
	return execWithoutKey(s, ctx,
		cts.MethodCreateKey,
		cts.MethodScopes[cts.MethodCreateKey],
		func(ctx context.Context) (*pk.CreateKeyResponse, error) {
			return s.keyService.CreateKey(ctx, req)
		},
	)
}

func (s *PolykeyService) ListKeys(ctx context.Context, req *pk.ListKeysRequest) (*pk.ListKeysResponse, error) {
	return execWithoutKey(s, ctx,
		cts.MethodListKeys,
		cts.MethodScopes[cts.MethodListKeys],
		func(ctx context.Context) (*pk.ListKeysResponse, error) {
			return s.keyService.ListKeys(ctx, req)
		},
	)
}

func (s *PolykeyService) RotateKey(ctx context.Context, req *pk.RotateKeyRequest) (*pk.RotateKeyResponse, error) {
	return execWithAuth(s, ctx,
		cts.MethodRotateKey,
		cts.MethodScopes[cts.MethodRotateKey],
		req.GetKeyId(),
		func(ctx context.Context, keyID domain.KeyID) (*pk.RotateKeyResponse, error) {
			return s.keyService.RotateKey(ctx, req)
		},
	)
}

func (s *PolykeyService) RevokeKey(ctx context.Context, req *pk.RevokeKeyRequest) (*emptypb.Empty, error) {
	return execWithAuth(s, ctx,
		cts.MethodRevokeKey,
		cts.MethodScopes[cts.MethodRevokeKey],
		req.GetKeyId(),
		func(ctx context.Context, keyID domain.KeyID) (*emptypb.Empty, error) {
			return &emptypb.Empty{}, s.keyService.RevokeKey(ctx, req)
		},
	)
}

func (s *PolykeyService) UpdateKeyMetadata(ctx context.Context, req *pk.UpdateKeyMetadataRequest) (*emptypb.Empty, error) {
	return execWithAuth(s, ctx,
		cts.MethodUpdateKeyMetadata,
		cts.MethodScopes[cts.MethodUpdateKeyMetadata],
		req.GetKeyId(),
		func(ctx context.Context, keyID domain.KeyID) (*emptypb.Empty, error) {
			return &emptypb.Empty{}, s.keyService.UpdateKeyMetadata(ctx, req)
		},
	)
}

func (s *PolykeyService) GetKeyMetadata(ctx context.Context, req *pk.GetKeyMetadataRequest) (*pk.GetKeyMetadataResponse, error) {
	return execWithAuth(s, ctx,
		cts.MethodGetKeyMetadata,
		cts.MethodScopes[cts.MethodGetKeyMetadata],
		req.GetKeyId(),
		func(ctx context.Context, keyID domain.KeyID) (*pk.GetKeyMetadataResponse, error) {
			return s.keyService.GetKeyMetadata(ctx, req)
		},
	)
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
