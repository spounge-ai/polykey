
package integration_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"log/slog"
	"testing"
	"time"

	app_grpc "github.com/spounge-ai/polykey/internal/app/grpc"
	app_errors "github.com/spounge-ai/polykey/internal/errors"
	infra_audit "github.com/spounge-ai/polykey/internal/infra/audit"
	"github.com/spounge-ai/polykey/internal/infra/auth"
	infra_config "github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/infra/persistence"
	"github.com/spounge-ai/polykey/internal/kms"
	"github.com/spounge-ai/polykey/internal/service"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func setupServer(t *testing.T) (pk.PolykeyServiceClient, func()) {
	truncate(t)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	cfg := &infra_config.Config{
		Server: infra_config.ServerConfig{
			Port: 0, // Dynamic port
			Mode: "test",
			RateLimiter: infra_config.RateLimiterConfig{
				Enabled: true,
				Rate:    1000,
				Burst:   1000,
			},
		},
		Authorization: infra_config.AuthorizationConfig{
			Roles: map[string]infra_config.RoleConfig{
				"user": {
					AllowedOperations: []string{"keys:create", "keys:read", "keys:update", "keys:revoke", "keys:list", "keys:rotate"},
				},
				"unauthorized": {
					AllowedOperations: []string{},
				},
			},
		},
		BootstrapSecrets: infra_config.BootstrapSecrets{
			PolykeyMasterKey: "/kH+AgL+tN2qrA8I+nXL7is4ORj23p2YVhpTjAz2YIs=",
			JWTRSAPrivateKey: string(privateKeyPEM),
		},
		DefaultKMSProvider:    "local",
		ClientCredentialsPath: "configs/dev_client/config.client.dev.yaml",
	}

	kmsProviders := make(map[string]kms.KMSProvider)
	localKMS, err := kms.NewLocalKMSProvider(cfg.BootstrapSecrets.PolykeyMasterKey)
	require.NoError(t, err)
	kmsProviders["local"] = localKMS

	keyRepo, err := persistence.NewPSQLAdapter(dbpool, slog.Default())
	require.NoError(t, err)

	auditRepo, err := persistence.NewAuditRepository(dbpool)
	require.NoError(t, err)
	auditLogger := infra_audit.NewAuditLogger(slog.Default(), auditRepo)

	authorizer := auth.NewAuthorizer(cfg.Authorization, keyRepo, auditLogger)

	clientStore, err := auth.NewFileClientStore(cfg.ClientCredentialsPath)
	require.NoError(t, err)

	tokenStore := auth.NewInMemoryTokenStore()
	tokenManager, err := auth.NewTokenManager(cfg.BootstrapSecrets.JWTRSAPrivateKey, tokenStore, auditLogger)
	require.NoError(t, err)

	keyService := service.NewKeyService(cfg, keyRepo, kmsProviders, slog.Default(), app_errors.NewErrorClassifier(slog.Default()), auditLogger)
	authService := service.NewAuthService(clientStore, tokenManager, 1*time.Hour)

	srv, port, err := app_grpc.New(cfg, keyService, authService, authorizer, auditLogger, slog.Default(), app_errors.NewErrorClassifier(slog.Default()), nil)
	require.NoError(t, err)

	go func() {
		if err := srv.Start(context.Background()); err != nil {
			log.Printf("Server exited with error: %v", err)
		}
	}()

	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%d", port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	client := pk.NewPolykeyServiceClient(conn)

	cleanup := func() {
		if err := conn.Close(); err != nil {
			t.Logf("failed to close connection: %v", err)
		}
		if err := srv.Stop(context.Background()); err != nil {
			t.Logf("failed to stop server: %v", err)
		}
	}

	return client, cleanup
}

func getAuthorizedContext(t *testing.T, client pk.PolykeyServiceClient) context.Context {
	authResp, err := client.Authenticate(context.Background(), &pk.AuthenticateRequest{
		ClientId: "polykey-dev-client",
		ApiKey:   "supersecretdevpassword",
	})
	require.NoError(t, err)
	return metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+authResp.AccessToken)
}

func TestHealthCheck(t *testing.T) {
	client, cleanup := setupServer(t)
	defer cleanup()

	ctx := getAuthorizedContext(t, client)

	resp, err := client.HealthCheck(ctx, &emptypb.Empty{})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, pk.HealthStatus_HEALTH_STATUS_HEALTHY, resp.Status)
}

func TestKeyLifecycle(t *testing.T) {
	client, cleanup := setupServer(t)
	defer cleanup()

	ctx := getAuthorizedContext(t, client)

	// Create
	createResp, err := client.CreateKey(ctx, &pk.CreateKeyRequest{
		KeyType:          pk.KeyType_KEY_TYPE_AES_256,
		RequesterContext: &pk.RequesterContext{ClientIdentity: "polykey-dev-client"},
	})
	require.NoError(t, err)
	require.NotNil(t, createResp)
	keyId := createResp.KeyId

	// Get
	getResp, err := client.GetKey(ctx, &pk.GetKeyRequest{KeyId: keyId, RequesterContext: &pk.RequesterContext{ClientIdentity: "polykey-dev-client"}})
	require.NoError(t, err)
	require.NotNil(t, getResp)
	require.Equal(t, keyId, getResp.Metadata.KeyId)

	// Update Metadata
	_, err = client.UpdateKeyMetadata(ctx, &pk.UpdateKeyMetadataRequest{KeyId: keyId, Description: &[]string{"new description"}[0], RequesterContext: &pk.RequesterContext{ClientIdentity: "polykey-dev-client"}})
	require.NoError(t, err)

	// Get Metadata
	getMetaResp, err := client.GetKeyMetadata(ctx, &pk.GetKeyMetadataRequest{KeyId: keyId, RequesterContext: &pk.RequesterContext{ClientIdentity: "polykey-dev-client"}})
	require.NoError(t, err)
	require.Equal(t, "new description", getMetaResp.Metadata.Description)

	// Rotate
	rotateResp, err := client.RotateKey(ctx, &pk.RotateKeyRequest{KeyId: keyId, RequesterContext: &pk.RequesterContext{ClientIdentity: "polykey-dev-client"}})
	require.NoError(t, err)
	require.Equal(t, int32(2), rotateResp.NewVersion)

	// Revoke
	_, err = client.RevokeKey(ctx, &pk.RevokeKeyRequest{KeyId: keyId, RequesterContext: &pk.RequesterContext{ClientIdentity: "polykey-dev-client"}})
	require.NoError(t, err)

	// Get after revoke should fail with specific status
	_, err = client.GetKey(ctx, &pk.GetKeyRequest{KeyId: keyId, RequesterContext: &pk.RequesterContext{ClientIdentity: "polykey-dev-client"}})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	// Note: The service currently doesn't prevent fetching revoked keys. This is a potential area for improvement.
	assert.NotEqual(t, codes.OK, st.Code())
}

func TestUnauthorized(t *testing.T) {
	client, cleanup := setupServer(t)
	defer cleanup()

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer invalid-token")

	_, err := client.CreateKey(ctx, &pk.CreateKeyRequest{})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Unauthenticated, st.Code())
}
