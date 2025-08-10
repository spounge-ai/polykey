package domain

import (
	"context"

	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// AuthenticatedUser represents a user that has been authenticated.
type AuthenticatedUser struct {
	ID   string
	Role string
}

type contextKey string

const (
	userContextKey = contextKey("user")
)

func NewContextWithUser(ctx context.Context, user *AuthenticatedUser) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFromContext(ctx context.Context) (*AuthenticatedUser, bool) {
	user, ok := ctx.Value(userContextKey).(*AuthenticatedUser)
	return user, ok
}

type Authorizer interface {
	Authorize(ctx context.Context, reqContext *pk.RequesterContext, attrs *pk.AccessAttributes, operation string, keyID KeyID) (bool, string)
}