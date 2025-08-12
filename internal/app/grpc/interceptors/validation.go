package interceptors

import (
	"context"

	"github.com/spounge-ai/polykey/internal/validation"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UnaryValidationInterceptor(validator *validation.RequestValidator) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		switch r := req.(type) {
		case *pk.CreateKeyRequest:
			if err := validator.ValidateCreateKeyRequest(ctx, r); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
			}
		}

		return handler(ctx, req)
	}
}
