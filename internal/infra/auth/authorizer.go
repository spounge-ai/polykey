package auth

import (
	"context"

	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// Authorizer defines the interface for authorization checks.
type Authorizer interface {
	Authorize(ctx context.Context, reqContext *pk.RequesterContext, attrs *pk.AccessAttributes, operation string) (bool, string)
}

// NewAuthorizer creates a new basic authorizer.
func NewAuthorizer() Authorizer {
	return &realAuthorizer{}
}

// realAuthorizer implements the Authorizer interface with simplified logic.
type realAuthorizer struct{}

// Authorize performs a simplified authorization check.
func (a *realAuthorizer) Authorize(ctx context.Context, reqContext *pk.RequesterContext, attrs *pk.AccessAttributes, operation string) (bool, string) {
	// This is a placeholder for a real authorization logic.
	// For now, it will always return true.
	return true, "mock_auth_decision_id_granted"
}
