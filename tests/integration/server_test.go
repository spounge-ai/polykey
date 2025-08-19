
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

func TestListKeys(t *testing.T) {
	client, cleanup := setupServer(t)
	defer cleanup()

	ctx := getAuthorizedContext(t, client)

	// Create some keys
	for i := 0; i < 5; i++ {
		_, err := client.CreateKey(ctx, &pk.CreateKeyRequest{
			KeyType:          pk.KeyType_KEY_TYPE_AES_256,
			RequesterContext: &pk.RequesterContext{ClientIdentity: "polykey-dev-client"},
		})
		require.NoError(t, err)
	}

	// List keys
	listResp, err := client.ListKeys(ctx, &pk.ListKeysRequest{
		PageSize:         10,
		RequesterContext: &pk.RequesterContext{ClientIdentity: "polykey-dev-client"},
	})
	require.NoError(t, err)
	require.NotNil(t, listResp)
	require.Len(t, listResp.Keys, 5)
}

func TestBatchOperations(t *testing.T) {
	client, cleanup := setupServer(t)
	defer cleanup()

	ctx := getAuthorizedContext(t, client)

	// Batch Create
	createItems := []*pk.CreateKeyItem{
		{KeyType: pk.KeyType_KEY_TYPE_AES_256, Description: "key 1"},
		{KeyType: pk.KeyType_KEY_TYPE_AES_256, Description: "key 2"},
	}
	batchCreateResp, err := client.BatchCreateKeys(ctx, &pk.BatchCreateKeysRequest{
		Keys:             createItems,
		RequesterContext: &pk.RequesterContext{ClientIdentity: "polykey-dev-client"},
	})
	require.NoError(t, err)
	require.NotNil(t, batchCreateResp)
	require.Len(t, batchCreateResp.Results, 2)
	require.True(t, batchCreateResp.Results[0].GetSuccess() != nil)
	require.True(t, batchCreateResp.Results[1].GetSuccess() != nil)

	keyId1 := batchCreateResp.Results[0].GetSuccess().KeyId
	keyId2 := batchCreateResp.Results[1].GetSuccess().KeyId

	// Batch Get
	batchGetResp, err := client.BatchGetKeys(ctx, &pk.BatchGetKeysRequest{
		Keys: []*pk.KeyRequestItem{
			{KeyId: keyId1},
			{KeyId: keyId2},
		},
		RequesterContext: &pk.RequesterContext{ClientIdentity: "polykey-dev-client"},
	})
	require.NoError(t, err)
	require.NotNil(t, batchGetResp)
	require.Len(t, batchGetResp.Results, 2)
	require.True(t, batchGetResp.Results[0].GetSuccess() != nil)
	require.True(t, batchGetResp.Results[1].GetSuccess() != nil)

	// Batch Get Metadata
	batchGetMetaResp, err := client.BatchGetKeyMetadata(ctx, &pk.BatchGetKeyMetadataRequest{
		Keys: []*pk.GetKeyMetadataItem{
			{KeyId: keyId1},
			{KeyId: keyId2},
		},
		RequesterContext: &pk.RequesterContext{ClientIdentity: "polykey-dev-client"},
	})
	require.NoError(t, err)
	require.NotNil(t, batchGetMetaResp)
	require.Len(t, batchGetMetaResp.Results, 2)
	require.True(t, batchGetMetaResp.Results[0].GetSuccess() != nil)
	require.True(t, batchGetMetaResp.Results[1].GetSuccess() != nil)

	// Batch Update Metadata
	desc1 := "new desc 1"
	desc2 := "new desc 2"
	batchUpdateMetaResp, err := client.BatchUpdateKeyMetadata(ctx, &pk.BatchUpdateKeyMetadataRequest{
		Keys: []*pk.UpdateKeyMetadataItem{
			{KeyId: keyId1, Description: &desc1},
			{KeyId: keyId2, Description: &desc2},
		},
		RequesterContext: &pk.RequesterContext{ClientIdentity: "polykey-dev-client"},
	})
	require.NoError(t, err)
	require.NotNil(t, batchUpdateMetaResp)
	require.Len(t, batchUpdateMetaResp.Results, 2)
	require.True(t, batchUpdateMetaResp.Results[0].GetSuccess())
	require.True(t, batchUpdateMetaResp.Results[1].GetSuccess())

	// Batch Rotate
	batchRotateResp, err := client.BatchRotateKeys(ctx, &pk.BatchRotateKeysRequest{
		Keys: []*pk.RotateKeyItem{
			{KeyId: keyId1},
			{KeyId: keyId2},
		},
		RequesterContext: &pk.RequesterContext{ClientIdentity: "polykey-dev-client"},
	})
	require.NoError(t, err)
	require.NotNil(t, batchRotateResp)
	require.Len(t, batchRotateResp.Results, 2)
	require.True(t, batchRotateResp.Results[0].GetSuccess() != nil)
	require.True(t, batchRotateResp.Results[1].GetSuccess() != nil)

	// Batch Revoke
	batchRevokeResp, err := client.BatchRevokeKeys(ctx, &pk.BatchRevokeKeysRequest{
		Keys: []*pk.RevokeKeyItem{
			{KeyId: keyId1},
			{KeyId: keyId2},
		},
		RequesterContext: &pk.RequesterContext{ClientIdentity: "polykey-dev-client"},
	})
	require.NoError(t, err)
	require.NotNil(t, batchRevokeResp)
	require.Len(t, batchRevokeResp.Results, 2)
	require.True(t, batchRevokeResp.Results[0].GetSuccess())
	require.True(t, batchRevokeResp.Results[1].GetSuccess())
}
