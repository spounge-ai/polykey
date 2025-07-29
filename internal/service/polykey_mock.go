package service

import (
	"context"
	"time"

	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MockPolykeyService implements the PolykeyService interface for testing purposes.
type MockPolykeyService struct{
	pk.UnimplementedPolykeyServiceServer
}

// GetKey is a mock implementation.
func (s *MockPolykeyService) GetKey(ctx context.Context, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error) {
	return nil, nil
}

// ListKeys is a mock implementation.
func (s *MockPolykeyService) ListKeys(ctx context.Context, req *pk.ListKeysRequest) (*pk.ListKeysResponse, error) {
	return nil, nil
}

// CreateKey is a mock implementation.
func (s *MockPolykeyService) CreateKey(ctx context.Context, req *pk.CreateKeyRequest) (*pk.CreateKeyResponse, error) {
	return nil, nil
}

// RotateKey is a mock implementation.
func (s *MockPolykeyService) RotateKey(ctx context.Context, req *pk.RotateKeyRequest) (*pk.RotateKeyResponse, error) {
	return nil, nil
}

// RevokeKey is a mock implementation.
func (s *MockPolykeyService) RevokeKey(ctx context.Context, req *pk.RevokeKeyRequest) (*emptypb.Empty, error) {
	return nil, nil
}

// UpdateKeyMetadata is a mock implementation.
func (s *MockPolykeyService) UpdateKeyMetadata(ctx context.Context, req *pk.UpdateKeyMetadataRequest) (*emptypb.Empty, error) {
	return nil, nil
}

// GetKeyMetadata is a mock implementation.
func (s *MockPolykeyService) GetKeyMetadata(ctx context.Context, req *pk.GetKeyMetadataRequest) (*pk.GetKeyMetadataResponse, error) {
	return nil, nil
}

// HealthCheck provides a mock health check response.
func (s *MockPolykeyService) HealthCheck(ctx context.Context, req *emptypb.Empty) (*pk.HealthCheckResponse, error) {
	return &pk.HealthCheckResponse{
		Status:    pk.HealthStatus_HEALTH_STATUS_HEALTHY,
		Timestamp: timestamppb.New(time.Now()),
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
		ServiceVersion: "v0.0.1-mock",
		BuildCommit:    "mock-commit-12345",
	}, nil
}

