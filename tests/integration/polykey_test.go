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

func setup(t *testing.T) (pk.PolykeyServiceClient, *auth.TokenManager, func(), context.Context) {
	truncate(t)

	// Generate RSA key for testing
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
		},
		Authorization: infra_config.AuthorizationConfig{
			Roles: map[string]infra_config.RoleConfig{
				"user": {
					AllowedOperations: []string{"create_key", "get_key"},
				},
				"unauthorized": {
					AllowedOperations: []string{},
				},
			},
		},
		BootstrapSecrets: infra_config.BootstrapSecrets{
			PolykeyMasterKey:   "/kH+AgL+tN2qrA8I+nXL7is4ORj23p2YVhpTjAz2YIs=",
			JWTRSAPrivateKey: string(privateKeyPEM),
		},
	}

	kmsProviders := make(map[string]kms.KMSProvider)
	localKMS, err := kms.NewLocalKMSProvider(cfg.BootstrapSecrets.PolykeyMasterKey)
	assert.NoError(t, err)
	kmsProviders["local"] = localKMS

	keyRepo, err := persistence.NewNeonDBStorage(dbpool)
	assert.NoError(t, err)

	authorizer := auth.NewAuthorizer(cfg.Authorization, keyRepo)

	keyService := service.NewKeyService(cfg, keyRepo, kmsProviders, slog.Default())

	srv, port, err := app_grpc.New(cfg, keyService, authorizer, nil, slog.Default()) // nil for audit logger for now
	assert.NoError(t, err)

	go func() {
		if err := srv.Start(); err != nil {
			log.Printf("Server exited with error: %v", err)
		}
	}()

	time.Sleep(2 * time.Second) // Give the server time to start

	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%d", port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	assert.NoError(t, err)

	client := pk.NewPolykeyServiceClient(conn)

	// Generate a JWT token for testing
	tokenManager, err := auth.NewTokenManager(cfg.BootstrapSecrets.JWTRSAPrivateKey)
	require.NoError(t, err)

	token, err := tokenManager.GenerateToken("test-user", []string{"user"}, time.Hour)
	assert.NoError(t, err)

	// Add the token to the context of the client
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)

	cleanup := func() {
		if err := conn.Close(); err != nil {
			t.Logf("failed to close connection: %v", err)
		}
		srv.Stop()
	}

	return client, tokenManager, cleanup, ctx
}

func TestHealthCheck(t *testing.T) {
	client, _, cleanup, ctx := setup(t)
	defer cleanup()

	resp, err := client.HealthCheck(ctx, &emptypb.Empty{})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, pk.HealthStatus_HEALTH_STATUS_HEALTHY, resp.Status)
}

func TestKeyOperations_HappyPath(t *testing.T) {
	client, _, cleanup, ctx := setup(t)
	defer cleanup()

	t.Run("CreateKey - Authorized", func(t *testing.T) {
		createReq := &pk.CreateKeyRequest{
			KeyType:          pk.KeyType_KEY_TYPE_AES_256,
			RequesterContext: &pk.RequesterContext{ClientIdentity: "test_creator"},
			InitialAuthorizedContexts: []string{"test-user"},
		}
		resp, err := client.CreateKey(ctx, createReq)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.KeyId)
	})

	t.Run("GetKey - Authorized", func(t *testing.T) {
		createReq := &pk.CreateKeyRequest{
			KeyType:          pk.KeyType_KEY_TYPE_AES_256,
			RequesterContext: &pk.RequesterContext{ClientIdentity: "test_creator"},
			InitialAuthorizedContexts: []string{"test-user"},
		}
		createResp, err := client.CreateKey(ctx, createReq)
		assert.NoError(t, err)

		resp, err := client.GetKey(ctx, &pk.GetKeyRequest{
			KeyId:            createResp.KeyId,
			RequesterContext: &pk.RequesterContext{ClientIdentity: "test_client"},
		})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, createResp.KeyId, resp.Metadata.KeyId)
	})
}

func TestKeyOperations_ErrorConditions(t *testing.T) {
	client, tokenManager, cleanup, ctx := setup(t)
	defer cleanup()

	t.Run("GetKey - Unauthorized", func(t *testing.T) {
		// Generate a JWT token for testing with a different role
		token, err := tokenManager.GenerateToken("test-user", []string{"unauthorized"}, time.Hour)
		assert.NoError(t, err)

		// Add the token to the context of the client
		ctxUnauth := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)

		createReq := &pk.CreateKeyRequest{
			KeyType:          pk.KeyType_KEY_TYPE_AES_256,
			RequesterContext: &pk.RequesterContext{ClientIdentity: "test_creator"},
			InitialAuthorizedContexts: []string{"test-user"},
		}
		createResp, err := client.CreateKey(ctx, createReq)
		assert.NoError(t, err)

		_, err = client.GetKey(ctxUnauth, &pk.GetKeyRequest{
			KeyId:            createResp.KeyId,
			RequesterContext: &pk.RequesterContext{ClientIdentity: "test_client"},
		})
		assert.Error(t, err)
		s, _ := status.FromError(err)
		assert.Equal(t, codes.PermissionDenied, s.Code())
	})
}
