package auth

import (
	"context"

	"github.com/spounge-ai/polykey/internal/domain"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

func NewAuthorizer() domain.Authorizer {
	return &realAuthorizer{}
}

// realAuthorizer implements the Authorizer interface with simplified logic.
type realAuthorizer struct{}

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

	switch operation {
	case "/polykey.v2.PolykeyService/CreateKey", "/polykey.v2.PolykeyService/RotateKey", "/polykey.v2.PolykeyService/RevokeKey", "/polykey.v2.PolykeyService/UpdateKeyMetadata":
		if role != "admin" {
			return false, "admin_required"
		}
	case "/polykey.v2.PolykeyService/GetKey", "/polykey.v2.PolykeyService/GetKeyMetadata", "/polykey.v2.PolykeyService/ListKeys":
		// All roles can perform these operations
	default:
		return false, "unknown_operation"
	}

	return true, "authorized"
}
