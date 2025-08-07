package grpc

import (
	"context"
	"log/slog"
	"time"

	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/service"
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
	service    service.KeyService
	authorizer domain.Authorizer
	audit      domain.AuditLogger
	logger     *slog.Logger
}

// NewPolykeyService creates a new instance of PolykeyService.
func NewPolykeyService(cfg *config.Config, service service.KeyService, authorizer domain.Authorizer, audit domain.AuditLogger, logger *slog.Logger) (pk.PolykeyServiceServer, error) {
	return &polykeyServiceImpl{
		cfg:        cfg,
		service:    service,
		authorizer: authorizer,
		audit:      audit,
		logger:     logger,
	}, nil
}

func (s *polykeyServiceImpl) GetKey(ctx context.Context, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error) {
	resp, err := s.service.GetKey(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get key: %v", err)
	}
	return resp, nil
}

func (s *polykeyServiceImpl) CreateKey(ctx context.Context, req *pk.CreateKeyRequest) (*pk.CreateKeyResponse, error) {
	resp, err := s.service.CreateKey(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create key: %v", err)
	}
	return resp, nil
}

func (s *polykeyServiceImpl) ListKeys(ctx context.Context, req *pk.ListKeysRequest) (*pk.ListKeysResponse, error) {
	resp, err := s.service.ListKeys(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list keys: %v", err)
	}
	return resp, nil
}

func (s *polykeyServiceImpl) RotateKey(ctx context.Context, req *pk.RotateKeyRequest) (*pk.RotateKeyResponse, error) {
	resp, err := s.service.RotateKey(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to rotate key: %v", err)
	}
	return resp, nil
}

func (s *polykeyServiceImpl) RevokeKey(ctx context.Context, req *pk.RevokeKeyRequest) (*emptypb.Empty, error) {
	if err := s.service.RevokeKey(ctx, req); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to revoke key: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (s *polykeyServiceImpl) UpdateKeyMetadata(ctx context.Context, req *pk.UpdateKeyMetadataRequest) (*emptypb.Empty, error) {
	if err := s.service.UpdateKeyMetadata(ctx, req); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update key metadata: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (s *polykeyServiceImpl) GetKeyMetadata(ctx context.Context, req *pk.GetKeyMetadataRequest) (*pk.GetKeyMetadataResponse, error) {
	resp, err := s.service.GetKeyMetadata(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get key metadata: %v", err)
	}
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
