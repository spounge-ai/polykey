package interceptors

import (
	"context"
	"strings"

	"github.com/spounge-ai/polykey/internal/domain"
	auth "github.com/spounge-ai/polykey/internal/infra/auth"
	"github.com/spounge-ai/polykey/internal/infra/ratelimit"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

var unprotectedMethods = map[string]struct{}{
	"/polykey.v2.PolykeyService/HealthCheck":   {},
	"/polykey.v2.PolykeyService/Authenticate": {},
}

// AuthenticationInterceptor validates the JWT token, extracts peer TLS info, and applies rate limiting.
func AuthenticationInterceptor(tokenManager *auth.TokenManager, limiter ratelimit.Limiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if _, isUnprotected := unprotectedMethods[info.FullMethod]; isUnprotected {
			return handler(ctx, req)
		}

		// Extract peer certificate information for zero-trust validation.
		if p, ok := peer.FromContext(ctx); ok {
			if tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo); ok {
				if len(tlsInfo.State.PeerCertificates) > 0 {
					// Add the leaf certificate to the context for the authorizer to use.
					ctx = domain.NewContextWithPeerCert(ctx, tlsInfo.State.PeerCertificates[0])
				}
			}
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

		claims, err := tokenManager.ValidateToken(ctx, token)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		// Apply rate limiting based on the client ID from the token.
		if !limiter.Allow(claims.UserID) {
			return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded for client %s", claims.UserID)
		}

		user := &domain.AuthenticatedUser{
			ID:          claims.UserID,
			Permissions: claims.Roles,
		}

		ctx = domain.NewContextWithUser(ctx, user)

		return handler(ctx, req)
	}
}
