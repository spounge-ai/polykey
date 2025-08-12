package domain

import (
	"context"

	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// AuthenticatedUser represents a user that has been authenticated.
// It contains the user's ID and a list of permissions.
type AuthenticatedUser struct {
	ID          string
	Permissions []string
	Tier        KeyTier
}

type contextKey string

const (
	userContextKey = contextKey("user")
)

// NewContextWithUser creates a new context with the authenticated user.
func NewContextWithUser(ctx context.Context, user *AuthenticatedUser) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// UserFromContext retrieves the authenticated user from the context.
func UserFromContext(ctx context.Context) (*AuthenticatedUser, bool) {
	user, ok := ctx.Value(userContextKey).(*AuthenticatedUser)
	return user, ok
}

// Authorizer defines the interface for an authorization service.
type Authorizer interface {
	Authorize(ctx context.Context, reqContext *pk.RequesterContext, attrs *pk.AccessAttributes, operation string, keyID KeyID) (bool, string)
}
