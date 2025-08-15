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

var unprotectedMethods = map[string]struct{}{
	"/polykey.v2.PolykeyService/HealthCheck":   {},
	"/polykey.v2.PolykeyService/Authenticate": {},
}

// AuthenticationInterceptor validates the JWT token from the request metadata
// for all RPCs except for a predefined list of unprotected methods.
func AuthenticationInterceptor(tokenManager *auth.TokenManager) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if _, isUnprotected := unprotectedMethods[info.FullMethod]; isUnprotected {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "metadata is not provided")
		}

		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			return nil, status.Error(codes.Unauthenticated, "authorization token is not provided")
		}

		authHeader := authHeaders[0]
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			return nil, status.Error(codes.Unauthenticated, "authorization header must use Bearer scheme")
		}

		token := authHeader[len(bearerPrefix):]
		if token == "" {
			return nil, status.Error(codes.Unauthenticated, "bearer token is empty")
		}

		claims, err := tokenManager.ValidateToken(token)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		user := &domain.AuthenticatedUser{
			ID:          claims.UserID,
			Permissions: claims.Roles,
			Tier:        claims.Tier,
		}

		ctx = domain.NewContextWithUser(ctx, user)

		return handler(ctx, req)
	}
}