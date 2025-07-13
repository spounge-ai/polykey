package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/SpoungeAI/polykey-service/internal/config"
	pk "github.com/spoungeai/spounge-proto/gen/go/polykey/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

func main() {
	log.Println("Starting dev_client...")

	// Load configuration
	loader := config.NewConfigLoader()
	cfg, err := loader.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Runtime: %s, Server: %s",
		loader.Detector.DetectRuntime(), cfg.ServerAddress)

	// Test network connectivity first
	tester := config.NewNetworkTester()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	if err := tester.TestConnection(ctx, cfg.ServerAddress); err != nil {
		log.Fatalf("Network test failed: %v", err)
	}

	// Create gRPC connection with modern approach
	conn, err := createGRPCConnection(cfg)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pk.NewPolykeyServiceClient(conn)

	// Execute test request
	if err := executeTestRequest(ctx, client); err != nil {
		log.Fatalf("Test request failed: %v", err)
	}

	log.Println("dev_client finished successfully.")
}

func createGRPCConnection(cfg *config.Config) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// Modern gRPC connection with proper options
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(4*1024*1024), // 4MB
			grpc.MaxCallSendMsgSize(4*1024*1024), // 4MB
		),
	}

	// Use DialContext instead of NewClient for better control
	conn, err := grpc.DialContext(ctx, cfg.ServerAddress, opts...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func executeTestRequest(ctx context.Context, client pk.PolykeyServiceClient) error {
	params, err := structpb.NewStruct(map[string]interface{}{
		"example_param": "value",
	})
	if err != nil {
		return err
	}

	req := &pk.ExecuteToolRequest{
		ToolName:   "example_tool",
		Parameters: params,
		UserId:     "user-123",
	}

	log.Printf("Calling ExecuteTool with tool_name: %s", req.ToolName)

	resp, err := client.ExecuteTool(ctx, req)
	if err != nil {
		return err
	}

	log.Printf("ExecuteTool status: %v", resp.Status)

	switch output := resp.Output.(type) {
	case *pk.ExecuteToolResponse_StringOutput:
		log.Printf("String Output: %s", output.StringOutput)
	case *pk.ExecuteToolResponse_StructOutput:
		log.Printf("Struct Output: %v", output.StructOutput)
	case *pk.ExecuteToolResponse_FileOutput:
		log.Printf("File Output: %+v", output.FileOutput)
	default:
		log.Println("No output returned")
	}

	return nil
}