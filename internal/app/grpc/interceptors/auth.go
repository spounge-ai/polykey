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
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if info.FullMethod == "/polykey.v2.PolykeyService/HealthCheck" {
			return handler(ctx, req)
		}

		var reqContext *pk.RequesterContext
		var attrs *pk.AccessAttributes

		switch r := req.(type) {
		case *pk.CreateKeyRequest:
			{
				reqContext = r.GetRequesterContext()
			}
		case *pk.GetKeyRequest:
			{
				reqContext = r.GetRequesterContext()
				attrs = &pk.AccessAttributes{Environment: r.GetKeyId()}
			}
		case *pk.GetKeyMetadataRequest:
			{
				reqContext = r.GetRequesterContext()
				attrs = &pk.AccessAttributes{Environment: r.GetKeyId()}
			}
		default:
			{
				return nil, status.Errorf(codes.Unimplemented, "unsupported request type: %T", req)
			}
		}

		if reqContext == nil {
			return nil, status.Errorf(codes.Unauthenticated, "missing requester context")
		}

		isAuthorized, reason := authorizer.Authorize(ctx, reqContext, attrs, info.FullMethod)
		if !isAuthorized {
			return nil, status.Errorf(codes.PermissionDenied, "permission denied: %s", reason)
		}

		return handler(ctx, req)
	}
}