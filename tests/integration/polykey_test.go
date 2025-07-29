package integration_test

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/spounge-ai/polykey/internal/config"
	"github.com/spounge-ai/polykey/internal/server"
	pb "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

func TestHealthCheck(t *testing.T) {
	// Load configuration
	cfg, err := config.Load("") // Load default config or provide a path
	assert.NoError(t, err)

	// Set a test port to 0 to get a dynamically assigned port
	cfg.Server.Port = 0 

	// Create and start the server in a goroutine
	srv, port, err := server.New(cfg)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := srv.Run(ctx); err != nil {
			log.Printf("Server exited with error: %v", err)
		}
	}()

	// Give the server a moment to start
	time.Sleep(2 * time.Second)

	// Set up a connection to the gRPC server using the dynamically assigned port
	conn, err := grpc.Dial(fmt.Sprintf("localhost:%d", port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	defer conn.Close()

	client := pb.NewPolykeyServiceClient(conn)

	// Call the HealthCheck method
	resp, err := client.HealthCheck(context.Background(), &emptypb.Empty{})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, pb.HealthStatus_HEALTH_STATUS_HEALTHY, resp.Status)
	assert.Equal(t, "1.0.0", resp.ServiceVersion) // Check against mock version
}

func TestGetKey(t *testing.T) {
	// Load configuration
	cfg, err := config.Load("") // Load default config or provide a path
	assert.NoError(t, err)

	// Set a test port to 0 to get a dynamically assigned port
	cfg.Server.Port = 0 

	// Create and start the server in a goroutine
	srv, port, err := server.New(cfg)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := srv.Run(ctx); err != nil {
			log.Printf("Server exited with error: %v", err)
		}
	}()

	// Give the server a moment to start
	time.Sleep(2 * time.Second)

	// Set up a connection to the gRPC server using the dynamically assigned port
	conn, err := grpc.Dial(fmt.Sprintf("localhost:%d", port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	defer conn.Close()

	client := pb.NewPolykeyServiceClient(conn)

	// Call the GetKey method
	keyID := "test_key_123"
	resp, err := client.GetKey(context.Background(), &pb.GetKeyRequest{
		KeyId: keyID,
		RequesterContext: &pb.RequesterContext{ClientIdentity: "test_client"},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, keyID, resp.Metadata.KeyId)
	assert.NotNil(t, resp.KeyMaterial.EncryptedKeyData)
}

func TestCreateKey(t *testing.T) {
	// Load configuration
	cfg, err := config.Load("") // Load default config or provide a path
	assert.NoError(t, err)

	// Set a test port to 0 to get a dynamically assigned port
	cfg.Server.Port = 0 

	// Create and start the server in a goroutine
	srv, port, err := server.New(cfg)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := srv.Run(ctx); err != nil {
			log.Printf("Server exited with error: %v", err)
		}
	}()

	// Give the server a moment to start
	time.Sleep(2 * time.Second)

	// Set up a connection to the gRPC server using the dynamically assigned port
	conn, err := grpc.Dial(fmt.Sprintf("localhost:%d", port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	defer conn.Close()

	client := pb.NewPolykeyServiceClient(conn)

	// Call the CreateKey method
	resp, err := client.CreateKey(context.Background(), &pb.CreateKeyRequest{
		KeyType: pb.KeyType_KEY_TYPE_AES_256,
		RequesterContext: &pb.RequesterContext{ClientIdentity: "test_creator"},
		Description: "Test AES-256 key",
		Tags: map[string]string{"purpose": "test"},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.KeyId)
	assert.Equal(t, pb.KeyType_KEY_TYPE_AES_256, resp.Metadata.KeyType)
}
