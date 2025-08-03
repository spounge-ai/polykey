package interceptors

import (
	"context"

	"github.com/spounge-ai/polykey/internal/domain"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryAuthInterceptor is a gRPC unary interceptor that performs authorization.
func UnaryAuthInterceptor(authorizer domain.Authorizer) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		var reqContext *pk.RequesterContext
		var attrs *pk.AccessAttributes

		// Extract common fields from the request.
		switch r := req.(type) {
		case *pk.CreateKeyRequest:
			reqContext = r.GetRequesterContext()
		case *pk.GetKeyRequest:
			reqContext = r.GetRequesterContext()
			attrs = &pk.AccessAttributes{Environment: r.KeyId}
		case *pk.GetKeyMetadataRequest:
			reqContext = r.GetRequesterContext()
		}

		isAuthorized, _ := authorizer.Authorize(ctx, reqContext, attrs, info.FullMethod)
		if !isAuthorized {
			return nil, status.Errorf(codes.PermissionDenied, "permission denied")
		}

		return handler(ctx, req)
	}
}
