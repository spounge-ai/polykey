
package integration_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"testing"
	"time"

	"github.com/spounge-ai/polykey/internal/domain"
	infra_audit "github.com/spounge-ai/polykey/internal/infra/audit"
	"github.com/spounge-ai/polykey/internal/infra/auth"
	"github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/infra/persistence"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"github.com/stretchr/testify/require"
)

func setupAuth(t *testing.T) (*auth.TokenManager, domain.Authorizer, domain.KeyRepository, func()) {
	truncate(t)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	cfg := &config.Config{
		Authorization: config.AuthorizationConfig{
			Roles: map[string]config.RoleConfig{
				"user": {
					AllowedOperations: []string{"keys:read"},
				},
				"admin": {
					AllowedOperations: []string{"*"},
				},
			},
		},
		BootstrapSecrets: config.BootstrapSecrets{
			JWTRSAPrivateKey: string(privateKeyPEM),
		},
	}

	keyRepo, err := persistence.NewPSQLAdapter(dbpool, slog.Default())
	require.NoError(t, err)

	auditRepo, err := persistence.NewAuditRepository(dbpool)
	require.NoError(t, err)
	auditLogger := infra_audit.NewAuditLogger(slog.Default(), auditRepo)

	authorizer := auth.NewAuthorizer(cfg.Authorization, keyRepo, auditLogger)

	tokenStore := auth.NewInMemoryTokenStore()
	tokenManager, err := auth.NewTokenManager(cfg.BootstrapSecrets.JWTRSAPrivateKey, tokenStore, auditLogger)
	require.NoError(t, err)

	return tokenManager, authorizer, keyRepo, func() {}
}

func TestTokenManager(t *testing.T) {
	tokenManager, _, _, cleanup := setupAuth(t)
	defer cleanup()

	userID := "test-user"
	roles := []string{"user"}

	token, err := tokenManager.GenerateToken(userID, roles, time.Hour)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := tokenManager.ValidateToken(context.Background(), token)
	require.NoError(t, err)
	require.Equal(t, userID, claims.UserID)
	require.Equal(t, roles, claims.Roles)

	err = tokenManager.Revoke(context.Background(), token)
	require.NoError(t, err)

	_, err = tokenManager.ValidateToken(context.Background(), token)
	require.Error(t, err)
}

func TestAuthorizer(t *testing.T) {
	_, authorizer, keyRepo, cleanup := setupAuth(t)
	defer cleanup()

	user := &domain.AuthenticatedUser{ID: "test-user", Permissions: []string{"user"}}
	admin := &domain.AuthenticatedUser{ID: "admin-user", Permissions: []string{"admin"}}

	ctxUser := domain.NewContextWithUser(context.Background(), user)
	ctxAdmin := domain.NewContextWithUser(context.Background(), admin)

	keyID := domain.NewKeyID()
	key := &domain.Key{
		ID:      keyID,
		Version: 1,
		Metadata: &pk.KeyMetadata{
			Description:        "test key",
			KeyType:            pk.KeyType_KEY_TYPE_AES_256,
			AuthorizedContexts: []string{"test-user"},
		},
		EncryptedDEK: []byte("encrypted-dek"),
		Status:       domain.KeyStatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err := keyRepo.CreateKey(context.Background(), key)
	require.NoError(t, err)

	// User with 'user' role can read
	allowed, reason := authorizer.Authorize(ctxUser, &pk.RequesterContext{ClientIdentity: "test-user"}, nil, "keys:read", keyID)
	require.True(t, allowed, reason)

	// User with 'user' role cannot create
	allowed, reason = authorizer.Authorize(ctxUser, &pk.RequesterContext{ClientIdentity: "test-user"}, nil, "keys:create", domain.KeyID{})
	require.False(t, allowed, reason)

	// Admin can do anything
	allowed, reason = authorizer.Authorize(ctxAdmin, &pk.RequesterContext{ClientIdentity: "admin-user"}, nil, "keys:create", domain.KeyID{})
	require.True(t, allowed, reason)

	allowed, reason = authorizer.Authorize(ctxAdmin, &pk.RequesterContext{ClientIdentity: "admin-user"}, nil, "keys:read", keyID)
	require.True(t, allowed, reason)
}
