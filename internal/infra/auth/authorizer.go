package auth

import (
	"context"
	"slices"

	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/internal/infra/config"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// NewAuthorizer creates a new authorizer.
// Note: The config is currently unused as permissions are derived directly from the JWT.
// It is kept for potential future use cases involving static role definitions.
func NewAuthorizer(cfg config.AuthorizationConfig, keyRepo domain.KeyRepository) domain.Authorizer {
	return &realAuthorizer{cfg: cfg, keyRepo: keyRepo}
}

type realAuthorizer struct {
	cfg     config.AuthorizationConfig
	keyRepo domain.KeyRepository
}

// Authorize checks if the authenticated user in the context is permitted to perform the given operation.
func (a *realAuthorizer) Authorize(ctx context.Context, reqContext *pk.RequesterContext, attrs *pk.AccessAttributes, operation string, keyID domain.KeyID) (bool, string) {
	user, ok, reason := a.authenticateAndAuthorize(ctx, operation)
	if !ok {
		return false, reason
	}

	// This is an example of resource-based authorization, which should be expanded.
	// For now, it only checks if the user is in the key's authorized contexts for the 'get_key' operation.
	if operation == "get_key" {
		key, err := a.keyRepo.GetKey(ctx, keyID)
		if err != nil {
			return false, "key_not_found"
		}

		if !slices.Contains(key.Metadata.AuthorizedContexts, user.ID) {
			return false, "insufficient_key_permissions"
		}
	}

	return true, "authorized"
}

// authenticateAndAuthorize checks the user's permissions from the context against the required operation.
func (a *realAuthorizer) authenticateAndAuthorize(ctx context.Context, operation string) (*domain.AuthenticatedUser, bool, string) {
	user, ok := domain.UserFromContext(ctx)
	if !ok {
		return nil, false, "missing_user_identity"
	}

	// Check for wildcard admin permission first.
	if slices.Contains(user.Permissions, "*") {
		return user, true, "authorized"
	}

	// Check for the specific operation permission.
	if !slices.Contains(user.Permissions, operation) {
		return nil, false, "operation_not_allowed"
	}

	return user, true, "authorized"
}