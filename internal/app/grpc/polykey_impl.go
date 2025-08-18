package grpc

import (
	"context"
	"fmt"
	"log/slog"
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


type PolykeyDeps struct {
	Config          *config.Config
	KeyService      service.KeyService
	AuthService     service.AuthService
	Authorizer      domain.Authorizer
	Audit           domain.AuditLogger
	Logger          *slog.Logger
	ErrorClassifier *app_errors.ErrorClassifier
}

type PolykeyService struct {
	pk.UnimplementedPolykeyServiceServer
	deps PolykeyDeps
}

func NewPolykeyService(deps PolykeyDeps) pk.PolykeyServiceServer {
	return &PolykeyService{deps: deps}
}

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

	if ok, reason := s.deps.Authorizer.Authorize(ctx, nil, nil, authOp, keyID); !ok {
		return zero, s.sanitizeError(ctx, methodName, fmt.Errorf("%w: %s", app_errors.ErrAuthorization, reason))
	}

	resp, err := fn(ctx, keyID)
	if err != nil {
		return zero, s.sanitizeError(ctx, methodName, err)
	}
	return resp, nil
}

func execWithoutKey[T any](
	s *PolykeyService,
	ctx context.Context,
	methodName, authOp string,
	fn func(context.Context) (T, error),
) (T, error) {
	var zero T

	if ok, reason := s.deps.Authorizer.Authorize(ctx, nil, nil, authOp, domain.KeyID{}); !ok {
		return zero, s.sanitizeError(ctx, methodName, fmt.Errorf("%w: %s", app_errors.ErrAuthorization, reason))
	}

	resp, err := fn(ctx)
	if err != nil {
		return zero, s.sanitizeError(ctx, methodName, err)
	}
	return resp, nil
}


func (s *PolykeyService) sanitizeError(ctx context.Context, method string, err error) error {
	return s.deps.ErrorClassifier.LogAndSanitize(ctx, s.deps.ErrorClassifier.Classify(err, method))
}

var emptyResponse = &emptypb.Empty{}


func toProtoTier(tier domain.KeyTier) cmn.ClientTier {
	switch tier {
	case domain.TierFree:
		return cmn.ClientTier_CLIENT_TIER_FREE
	case domain.TierPro:
		return cmn.ClientTier_CLIENT_TIER_PRO
	case domain.TierEnterprise:
		return cmn.ClientTier_CLIENT_TIER_ENTERPRISE
	default:
		return cmn.ClientTier_CLIENT_TIER_UNSPECIFIED
	}
}

func (s *PolykeyService) Authenticate(ctx context.Context, req *pk.AuthenticateRequest) (*pk.AuthenticateResponse, error) {
	if req.GetClientId() == "" || req.GetApiKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "client_id and api_key are required")
	}

	result, err := s.deps.AuthService.Authenticate(ctx, req.GetClientId(), req.GetApiKey())
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "authentication failed: %v", err)
	}

	return &pk.AuthenticateResponse{
		AccessToken: result.AccessToken,
		TokenType:   result.TokenType,
		ExpiresIn:   result.ExpiresIn,
		IssuedAt:    timestamppb.Now(),
		ClientTier:  toProtoTier(result.ClientTier),
	}, nil
}

func (s *PolykeyService) GetKey(ctx context.Context, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error) {
	return execWithAuth(s, ctx, cts.MethodGetKey, cts.MethodScopes[cts.MethodGetKey], req.GetKeyId(),
		func(ctx context.Context, keyID domain.KeyID) (*pk.GetKeyResponse, error) {
			return s.deps.KeyService.GetKey(ctx, req)
		})
}

func (s *PolykeyService) CreateKey(ctx context.Context, req *pk.CreateKeyRequest) (*pk.CreateKeyResponse, error) {
	return execWithoutKey(s, ctx, cts.MethodCreateKey, cts.MethodScopes[cts.MethodCreateKey],
		func(ctx context.Context) (*pk.CreateKeyResponse, error) {
			return s.deps.KeyService.CreateKey(ctx, req)
		})
}

func (s *PolykeyService) BatchCreateKeys(ctx context.Context, req *pk.BatchCreateKeysRequest) (*pk.BatchCreateKeysResponse, error) {
	return execWithoutKey(s, ctx, cts.MethodCreateKey, cts.MethodScopes[cts.MethodCreateKey],
		func(ctx context.Context) (*pk.BatchCreateKeysResponse, error) {
			return s.deps.KeyService.BatchCreateKeys(ctx, req)
		})
}

func (s *PolykeyService) BatchGetKeys(ctx context.Context, req *pk.BatchGetKeysRequest) (*pk.BatchGetKeysResponse, error) {
	return execWithoutKey(s, ctx, cts.MethodGetKey, cts.MethodScopes[cts.MethodGetKey],
		func(ctx context.Context) (*pk.BatchGetKeysResponse, error) {
			return s.deps.KeyService.BatchGetKeys(ctx, req)
		})
}

func (s *PolykeyService) BatchGetKeyMetadata(ctx context.Context, req *pk.BatchGetKeyMetadataRequest) (*pk.BatchGetKeyMetadataResponse, error) {
	return execWithoutKey(s, ctx, cts.MethodGetKeyMetadata, cts.MethodScopes[cts.MethodGetKeyMetadata],
		func(ctx context.Context) (*pk.BatchGetKeyMetadataResponse, error) {
			return s.deps.KeyService.BatchGetKeyMetadata(ctx, req)
		})
}

func (s *PolykeyService) BatchRotateKeys(ctx context.Context, req *pk.BatchRotateKeysRequest) (*pk.BatchRotateKeysResponse, error) {
	return execWithoutKey(s, ctx, cts.MethodRotateKey, cts.MethodScopes[cts.MethodRotateKey],
		func(ctx context.Context) (*pk.BatchRotateKeysResponse, error) {
			return s.deps.KeyService.BatchRotateKeys(ctx, req)
		})
}

func (s *PolykeyService) BatchRevokeKeys(ctx context.Context, req *pk.BatchRevokeKeysRequest) (*pk.BatchRevokeKeysResponse, error) {
	return execWithoutKey(s, ctx, cts.MethodRevokeKey, cts.MethodScopes[cts.MethodRevokeKey],
		func(ctx context.Context) (*pk.BatchRevokeKeysResponse, error) {
			return s.deps.KeyService.BatchRevokeKeys(ctx, req)
		})
}

func (s *PolykeyService) BatchUpdateKeyMetadata(ctx context.Context, req *pk.BatchUpdateKeyMetadataRequest) (*pk.BatchUpdateKeyMetadataResponse, error) {
	return execWithoutKey(s, ctx, cts.MethodUpdateKeyMetadata, cts.MethodScopes[cts.MethodUpdateKeyMetadata],
		func(ctx context.Context) (*pk.BatchUpdateKeyMetadataResponse, error) {
			return s.deps.KeyService.BatchUpdateKeyMetadata(ctx, req)
		})
}

func (s *PolykeyService) ListKeys(ctx context.Context, req *pk.ListKeysRequest) (*pk.ListKeysResponse, error) {
	return execWithoutKey(s, ctx, cts.MethodListKeys, cts.MethodScopes[cts.MethodListKeys],
		func(ctx context.Context) (*pk.ListKeysResponse, error) {
			return s.deps.KeyService.ListKeys(ctx, req)
		})
}

func (s *PolykeyService) RotateKey(ctx context.Context, req *pk.RotateKeyRequest) (*pk.RotateKeyResponse, error) {
	return execWithAuth(s, ctx, cts.MethodRotateKey, cts.MethodScopes[cts.MethodRotateKey], req.GetKeyId(),
		func(ctx context.Context, keyID domain.KeyID) (*pk.RotateKeyResponse, error) {
			return s.deps.KeyService.RotateKey(ctx, req)
		})
}

func (s *PolykeyService) RevokeKey(ctx context.Context, req *pk.RevokeKeyRequest) (*emptypb.Empty, error) {
	return execWithAuth(s, ctx, cts.MethodRevokeKey, cts.MethodScopes[cts.MethodRevokeKey], req.GetKeyId(),
		func(ctx context.Context, keyID domain.KeyID) (*emptypb.Empty, error) {
			return emptyResponse, s.deps.KeyService.RevokeKey(ctx, req)
		})
}

func (s *PolykeyService) UpdateKeyMetadata(ctx context.Context, req *pk.UpdateKeyMetadataRequest) (*emptypb.Empty, error) {
	return execWithAuth(s, ctx, cts.MethodUpdateKeyMetadata, cts.MethodScopes[cts.MethodUpdateKeyMetadata], req.GetKeyId(),
		func(ctx context.Context, keyID domain.KeyID) (*emptypb.Empty, error) {
			return emptyResponse, s.deps.KeyService.UpdateKeyMetadata(ctx, req)
		})
}

func (s *PolykeyService) GetKeyMetadata(ctx context.Context, req *pk.GetKeyMetadataRequest) (*pk.GetKeyMetadataResponse, error) {
	return execWithAuth(s, ctx, cts.MethodGetKeyMetadata, cts.MethodScopes[cts.MethodGetKeyMetadata], req.GetKeyId(),
		func(ctx context.Context, keyID domain.KeyID) (*pk.GetKeyMetadataResponse, error) {
			return s.deps.KeyService.GetKeyMetadata(ctx, req)
		})
}

func (s *PolykeyService) HealthCheck(ctx context.Context, req *emptypb.Empty) (*pk.HealthCheckResponse, error) {
	return &pk.HealthCheckResponse{
		Status:         pk.HealthStatus_HEALTH_STATUS_HEALTHY,
		Timestamp:      timestamppb.Now(),
		ServiceVersion: s.deps.Config.ServiceVersion,
		BuildCommit:    s.deps.Config.BuildCommit,
		Metrics: &pk.ServiceMetrics{
			UptimeSince: timestamppb.New(time.Now().Add(-24 * time.Hour)), // Mock uptime
		},
	}, nil
}
