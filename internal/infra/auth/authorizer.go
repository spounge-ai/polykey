package auth

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/spounge-ai/polykey/internal/constants"
	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/infra/persistence"
	"github.com/spounge-ai/polykey/pkg/cache"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("github.com/spounge-ai/polykey/internal/infra/auth")

// NewAuthorizer creates a new authorizer.
func NewAuthorizer(cfg config.AuthorizationConfig, keyRepo domain.KeyRepository) domain.Authorizer {
	return &realAuthorizer{
		cfg:     cfg,
		keyRepo: keyRepo,
		policyCache: cache.New[string, bool](
			cache.WithDefaultTTL[string, bool](5*time.Minute),
			cache.WithCleanupInterval[string, bool](10*time.Minute),
		),
	}
}

type realAuthorizer struct {
	cfg         config.AuthorizationConfig
	keyRepo     domain.KeyRepository
	policyCache cache.Store[string, bool]
}

func (a *realAuthorizer) getCacheKey(userID, operation string, keyID domain.KeyID) string {
	return fmt.Sprintf("%s:%s:%s", userID, operation, keyID.String())
}

// Authorize checks if the authenticated user in the context is permitted to perform the given operation.
func (a *realAuthorizer) Authorize(ctx context.Context, reqContext *pk.RequesterContext, attrs *pk.AccessAttributes, operation string, keyID domain.KeyID) (bool, string) {
	ctx, span := tracer.Start(ctx, "Authorize", trace.WithAttributes(
		attribute.String("auth.operation", operation),
		attribute.String("auth.key_id", keyID.String()),
	))
	defer span.End()

	user, ok, reason := a.authenticateAndAuthorize(ctx, operation)
	if !ok {
		span.SetAttributes(attribute.Bool("auth.authorized", false), attribute.String("auth.reason", reason))
		return false, reason
	}
	span.SetAttributes(attribute.String("auth.user_id", user.ID))

	cacheKey := a.getCacheKey(user.ID, operation, keyID)
	if authorized, found := a.policyCache.Get(ctx, cacheKey);
 found {
		span.SetAttributes(attribute.Bool("auth.cache_hit", true))
		if !authorized {
			return false, "operation_not_allowed_by_cache"
		}
		return true, "authorized_by_cache"
	}

	span.SetAttributes(attribute.Bool("auth.cache_hit", false))

	authorized, reason := a.checkAuthorization(ctx, user, operation, keyID)
	if authorized {
		a.policyCache.Set(ctx, cacheKey, true, 0) // Use default TTL
		span.SetAttributes(attribute.Bool("auth.authorized", true), attribute.String("auth.reason", reason))
	} else {
		span.SetAttributes(attribute.Bool("auth.authorized", false), attribute.String("auth.reason", reason))
	}

	return authorized, reason
}

func (a *realAuthorizer) checkAuthorization(ctx context.Context, user *domain.AuthenticatedUser, operation string, keyID domain.KeyID) (bool, string) {
	// If keyID is not provided, we can't do resource-based authorization.
	// This applies to operations like CreateKey or ListKeys.
	if keyID.IsZero() {
		return true, "authorized"
	}

	// For operations on a specific key, perform resource-based authorization.
	switch operation {
	case constants.AuthKeysRead, constants.AuthKeysRotate, constants.AuthKeysRevoke, constants.AuthKeysUpdate:
		key, err := a.keyRepo.GetKey(ctx, keyID)
		if err != nil {
			if errors.Is(err, persistence.ErrKeyNotFound) {
				return false, "key_not_found"
			}
			// For other errors, it's better to not leak details.
			return false, "internal_error_accessing_key"
		}

		if key.Metadata == nil {
			return false, "key_missing_metadata"
		}

		// Check if user is in the key's authorized contexts.
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
