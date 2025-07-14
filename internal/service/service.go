package service

import (
	"context"

	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v1"
)

type Service interface {
	ExecuteTool(ctx context.Context, req *pk.ExecuteToolRequest) (*pk.ExecuteToolResponse, error)
	ExecuteToolStream(req *pk.ExecuteToolStreamRequest, stream pk.PolykeyService_ExecuteToolStreamServer) error
}