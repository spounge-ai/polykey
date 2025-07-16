// File: internal/service/service.go
package service

import (
	"context"

	cmn "github.com/spounge-ai/spounge-proto/gen/go/common/v2"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/protobuf/types/known/structpb"
)

// Service interface that both server and mock implement
type Service interface {
	ExecuteTool(ctx context.Context, toolName string, parameters *structpb.Struct, secretId *string, metadata *cmn.Metadata) (*pk.ExecuteToolResponse, error)
}