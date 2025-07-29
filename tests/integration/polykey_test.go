package integration_test

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/spounge-ai/polykey/internal/config"
	"github.com/spounge-ai/polykey/internal/server"
	pb "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// setupTestServer starts a new server and returns a client connection and a cleanup function.
func setupTestServer(t *testing.T) (pb.PolykeyServiceClient, func()) {
	cfg, err := config.Load("")
	assert.NoError(t, err)
	cfg.Server.Port = 0 // Use a dynamic port for testing

	srv, port, err := server.New(cfg)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		if err := srv.Run(ctx); err != nil {
			log.Printf("Server exited with error: %v", err)
		}
	}()

	time.Sleep(2 * time.Second) // Give the server time to start

	conn, err := grpc.Dial(fmt.Sprintf("localhost:%d", port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)

	client := pb.NewPolykeyServiceClient(conn)

	cleanup := func() {
		conn.Close()
		cancel()
	}

	return client, cleanup
}

func TestHealthCheck(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	resp, err := client.HealthCheck(context.Background(), &emptypb.Empty{})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, pb.HealthStatus_HEALTH_STATUS_HEALTHY, resp.Status)
}

func TestKeyOperations_HappyPath(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	t.Run("GetKey - Authorized", func(t *testing.T) {
		keyID := "test_key_123"
		resp, err := client.GetKey(context.Background(), &pb.GetKeyRequest{
			KeyId:            keyID,
			RequesterContext: &pb.RequesterContext{ClientIdentity: "test_client"},
		})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, keyID, resp.Metadata.KeyId)
	})

	t.Run("CreateKey - Authorized", func(t *testing.T) {
		createReq := &pb.CreateKeyRequest{
			KeyType:          pb.KeyType_KEY_TYPE_AES_256,
			RequesterContext: &pb.RequesterContext{ClientIdentity: "test_creator"},
		}
		resp, err := client.CreateKey(context.Background(), createReq)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.KeyId)
	})

	t.Run("GetKeyMetadata - Authorized", func(t *testing.T) {
		keyID := "test_key_for_metadata"
		resp, err := client.GetKeyMetadata(context.Background(), &pb.GetKeyMetadataRequest{
			KeyId:                keyID,
			RequesterContext:     &pb.RequesterContext{ClientIdentity: "test_client"},
			IncludeAccessHistory: true,
			IncludePolicyDetails: true,
		})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, keyID, resp.Metadata.KeyId)
		assert.NotEmpty(t, resp.AccessHistory)
		assert.NotEmpty(t, resp.PolicyDetails)
	})
}

func TestKeyOperations_ErrorConditions(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	t.Run("GetKey - Unauthorized", func(t *testing.T) {
		keyID := "restricted_key"
		_, err := client.GetKey(context.Background(), &pb.GetKeyRequest{
			KeyId:            keyID,
			RequesterContext: &pb.RequesterContext{ClientIdentity: "test_client"},
		})
		assert.Error(t, err)
		s, _ := status.FromError(err)
		assert.Equal(t, codes.PermissionDenied, s.Code())
	})

	t.Run("CreateKey - Unauthorized", func(t *testing.T) {
		_, err := client.CreateKey(context.Background(), &pb.CreateKeyRequest{
			KeyType:          pb.KeyType_KEY_TYPE_API_KEY,
			RequesterContext: &pb.RequesterContext{ClientIdentity: "unknown_creator"},
		})
		assert.Error(t, err)
		s, _ := status.FromError(err)
		assert.Equal(t, codes.PermissionDenied, s.Code())
	})

	t.Run("GetKeyMetadata - Unauthorized", func(t *testing.T) {
		keyID := "test_key_for_metadata"
		_, err := client.GetKeyMetadata(context.Background(), &pb.GetKeyMetadataRequest{
			KeyId:            keyID,
			RequesterContext: &pb.RequesterContext{ClientIdentity: "unknown_client"},
		})
		assert.Error(t, err)
		s, _ := status.FromError(err)
		assert.Equal(t, codes.PermissionDenied, s.Code())
	})
}