package auth

import (
	"context"

	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/internal/infra/config"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

func NewAuthorizer(cfg config.AuthorizationConfig) domain.Authorizer {
	return &realAuthorizer{cfg: cfg}
}

// realAuthorizer implements the Authorizer interface with simplified logic.
type realAuthorizer struct{
	cfg config.AuthorizationConfig
}

// Authorize performs a simplified authorization check.
func (a *realAuthorizer) Authorize(ctx context.Context, reqContext *pk.RequesterContext, attrs *pk.AccessAttributes, operation string) (bool, string) {
	if reqContext == nil || reqContext.ClientIdentity == "" {
		return false, "missing_client_identity"
	}

	// Simple RBAC model
	role := "user" // Default role
	if reqContext.ClientIdentity == "admin" {
		role = "admin"
	}

	roleConfig, ok := a.cfg.Roles[role]
	if !ok {
		return false, "invalid_role"
	}

	for _, allowedOperation := range roleConfig.AllowedOperations {
		if allowedOperation == operation {
			return true, "authorized"
		}
	}

	return false, "operation_not_allowed"
}
