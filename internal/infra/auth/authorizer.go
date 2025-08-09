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
	if reqContext == nil || reqContext.ClientIdentity == "" {
		return false, "missing_client_identity"
	}

	role := a.determineRole(reqContext.ClientIdentity)
	roleConfig, ok := a.cfg.Roles[role]
	if !ok {
		return false, "invalid_role"
	}

	if !slices.Contains(roleConfig.AllowedOperations, operation) {
		return false, "operation_not_allowed"
	}

	if a.isEmptyKeyID(keyID) {
		return true, "authorized"
	}

	return a.authorizeKeyAccess(ctx, reqContext.ClientIdentity, keyID)
}

func (a *realAuthorizer) determineRole(clientIdentity string) string {
	if clientIdentity == "admin" {
		return "admin"
	}
	return "user"
}

func (a *realAuthorizer) isEmptyKeyID(keyID domain.KeyID) bool {
	return keyID == domain.KeyID{} || keyID.String() == "00000000-0000-0000-0000-000000000000"
}

func (a *realAuthorizer) authorizeKeyAccess(ctx context.Context, clientIdentity string, keyID domain.KeyID) (bool, string) {
	key, err := a.keyRepo.GetKey(ctx, keyID)
	if err != nil {
		return false, "key_not_found"
	}

	if slices.Contains(key.Metadata.AuthorizedContexts, clientIdentity) {
		return true, "authorized"
	}

	return false, "insufficient_key_permissions"
}