package service

import (
	"context"

	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/protobuf/types/known/emptypb"
)

// PolykeyService defines the interface for the Polykey microservice.
type PolykeyService interface {
	GetKey(ctx context.Context, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error)
	ListKeys(ctx context.Context, req *pk.ListKeysRequest) (*pk.ListKeysResponse, error)
	CreateKey(ctx context.Context, req *pk.CreateKeyRequest) (*pk.CreateKeyResponse, error)
	RotateKey(ctx context.Context, req *pk.RotateKeyRequest) (*pk.RotateKeyResponse, error)
	RevokeKey(ctx context.Context, req *pk.RevokeKeyRequest) (*emptypb.Empty, error)
	UpdateKeyMetadata(ctx context.Context, req *pk.UpdateKeyMetadataRequest) (*emptypb.Empty, error)
	GetKeyMetadata(ctx context.Context, req *pk.GetKeyMetadataRequest) (*pk.GetKeyMetadataResponse, error)
	HealthCheck(ctx context.Context, req *emptypb.Empty) (*pk.HealthCheckResponse, error)
}