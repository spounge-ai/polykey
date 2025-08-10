package auth

import (
	"context"
	"slices"

	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/internal/infra/config"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

func NewAuthorizer(cfg config.AuthorizationConfig, keyRepo domain.KeyRepository) domain.Authorizer {
	return &realAuthorizer{cfg: cfg, keyRepo: keyRepo}
}

type realAuthorizer struct {
	cfg     config.AuthorizationConfig
	keyRepo domain.KeyRepository
}

func (a *realAuthorizer) Authorize(ctx context.Context, reqContext *pk.RequesterContext, attrs *pk.AccessAttributes, operation string, keyID domain.KeyID) (bool, string) {
	user, ok, reason := a.authenticateAndAuthorize(ctx, operation, keyID)
	if !ok {
		return false, reason
	}

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

func (a *realAuthorizer) authenticateAndAuthorize(ctx context.Context, operation string, keyID domain.KeyID) (*domain.AuthenticatedUser, bool, string) {
	user, ok := domain.UserFromContext(ctx)
	if !ok {
		return nil, false, "missing_user_identity"
	}

	roleConfig, ok := a.cfg.Roles[user.Role]
	if !ok {
		return nil, false, "invalid_role"
	}

	if !slices.Contains(roleConfig.AllowedOperations, operation) {
		return nil, false, "operation_not_allowed"
	}

	return user, true, "authorized"
}
