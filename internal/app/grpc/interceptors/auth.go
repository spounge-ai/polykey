package interceptors

import (
	"context"
	"strings"

	"github.com/spounge-ai/polykey/internal/domain"
	auth "github.com/spounge-ai/polykey/internal/infra/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var unprotectedMethods = map[string]bool{
	"/polykey.v2.PolykeyService/HealthCheck":   true,
	"/polykey.v2.PolykeyService/Authenticate": true,
}

// AuthenticationInterceptor validates the JWT token from the request metadata
// for all RPCs except for a predefined list of unprotected methods.
func AuthenticationInterceptor(tokenManager *auth.TokenManager) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if unprotectedMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "metadata is not provided")
		}

		authHeader := md.Get("authorization")
		if len(authHeader) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "authorization token is not provided")
		}

		token := strings.TrimPrefix(authHeader[0], "Bearer ")
		claims, err := tokenManager.ValidateToken(token)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		user := &domain.AuthenticatedUser{
			ID:          claims.UserID,
			Permissions: claims.Roles, 
		}

		ctx = domain.NewContextWithUser(ctx, user)

		return handler(ctx, req)
	}
}