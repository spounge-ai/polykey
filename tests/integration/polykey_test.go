package integration_test

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"testing"
	"time"

	app_grpc "github.com/spounge-ai/polykey/internal/app/grpc"
	infra_config "github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/service"
	"github.com/spounge-ai/polykey/tests/mocks/auth"
	"github.com/spounge-ai/polykey/tests/mocks/kms"
	"github.com/spounge-ai/polykey/tests/mocks/persistence"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// setupTestServer starts a new server and returns a client connection and a cleanup function.
func setupTestServer(t *testing.T) (pk.PolykeyServiceClient, func()) {
	// Create a mock config for testing
	cfg := &infra_config.Config{
		Server: infra_config.ServerConfig{
			Port: 0, // Dynamic port
			Mode: "test",
		},
		Database: infra_config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "testuser",
			Password: "testpassword",
			DBName:   "testdb",
			SSLMode:  "disable",
		},
		Vault: infra_config.VaultConfig{
			Address: "http://localhost:8200",
			Token:   "testtoken",
		},
		AWS: infra_config.AWSConfig{
			Region:    "us-east-1",
			S3Bucket:  "test-bucket",
			KMSKeyARN: "arn:aws:kms:us-east-1:123456789012:key/mrk-12345678901234567890123456789012",
		},
	}

	// Always use mocks in integration tests
	log.Println("Running in TEST environment: Using mock implementations.")
	kmsAdapter := kms.NewMockKMSAdapter()
	authorizer := auth.NewMockAuthorizer()
	keyRepo := persistence.NewMockS3Storage()
	keyService := service.NewKeyService(keyRepo, kmsAdapter, slog.Default())

	srv, port, err := app_grpc.New(cfg, keyService, authorizer, nil, slog.Default()) // nil for audit logger for now
	assert.NoError(t, err)

	_, cancelFunc := context.WithCancel(context.Background())

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

	cleanup := func() {
		if err := conn.Close(); err != nil {
			t.Logf("failed to close connection: %v", err)
		}
		cancelFunc()
	}

	return client, cleanup
}

func TestHealthCheck(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	resp, err := client.HealthCheck(context.Background(), &emptypb.Empty{})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, pk.HealthStatus_HEALTH_STATUS_HEALTHY, resp.Status)
}

func TestKeyOperations_HappyPath(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	t.Run("GetKey - Authorized", func(t *testing.T) {
		keyID := "test_key_123"
		resp, err := client.GetKey(context.Background(), &pk.GetKeyRequest{
			KeyId:            keyID,
			RequesterContext: &pk.RequesterContext{ClientIdentity: "test_client"},
		})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, keyID, resp.Metadata.KeyId)
	})

	t.Run("CreateKey - Authorized", func(t *testing.T) {
		createReq := &pk.CreateKeyRequest{
			KeyType:          pk.KeyType_KEY_TYPE_AES_256,
			RequesterContext: &pk.RequesterContext{ClientIdentity: "test_creator"},
		}
		resp, err := client.CreateKey(context.Background(), createReq)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.KeyId)
	})

	t.Run("GetKeyMetadata - Authorized", func(t *testing.T) {
		keyID := "test_key_for_metadata"
		resp, err := client.GetKeyMetadata(context.Background(), &pk.GetKeyMetadataRequest{
			KeyId:                keyID,
			RequesterContext:     &pk.RequesterContext{ClientIdentity: "test_client"},
			IncludeAccessHistory: true,
			IncludePolicyDetails: true,
		})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, keyID, resp.Metadata.KeyId)
		// assert.NotEmpty(t, resp.AccessHistory) // TODO: Enable when audit log is implemented
		// assert.NotEmpty(t, resp.PolicyDetails) // TODO: Enable when policy engine is implemented
	})
}

func TestKeyOperations_ErrorConditions(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	t.Run("GetKey - Unauthorized", func(t *testing.T) {
		keyID := "restricted_key"
		_, err := client.GetKey(context.Background(), &pk.GetKeyRequest{
			KeyId:            keyID,
			RequesterContext: &pk.RequesterContext{ClientIdentity: "test_client"},
		})
		assert.Error(t, err)
		s, _ := status.FromError(err)
		assert.Equal(t, codes.PermissionDenied, s.Code())
	})

	t.Run("CreateKey - Unauthorized", func(t *testing.T) {
		_, err := client.CreateKey(context.Background(), &pk.CreateKeyRequest{
			KeyType:          pk.KeyType_KEY_TYPE_API_KEY,
			RequesterContext: &pk.RequesterContext{ClientIdentity: "unknown_creator"},
		})
		assert.Error(t, err)
		s, _ := status.FromError(err)
		assert.Equal(t, codes.PermissionDenied, s.Code())
	})

	t.Run("GetKeyMetadata - Unauthorized", func(t *testing.T) {
		keyID := "test_key_for_metadata"
		_, err := client.GetKeyMetadata(context.Background(), &pk.GetKeyMetadataRequest{
			KeyId:            keyID,
			RequesterContext: &pk.RequesterContext{ClientIdentity: "unknown_client"},
		})
		assert.Error(t, err)
		s, _ := status.FromError(err)
		assert.Equal(t, codes.PermissionDenied, s.Code())
	})
}
