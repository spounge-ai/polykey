package interceptors

import (
	"context"

	"github.com/spounge-ai/polykey/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UnaryAuthInterceptor(authorizer domain.Authorizer) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// In a real implementation, you would extract the principal from the context
		// (e.g., from a JWT token).
		// For this example, we'll use a mock principal.
		principal := "mock_principal"

		if !authorizer.Authorize(ctx, nil, nil, info.FullMethod) {
			return nil, status.Errorf(codes.PermissionDenied, "not authorized")
		}

		return handler(ctx, req)
	}
}
