package main

import (
	"context"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

func main() {
	port := os.Getenv("POLYKEY_GRPC_PORT")
	if port == "" {
		port = "50051" // Default port
	}

	conn, err := grpc.Dial("localhost:" + port, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := pb.NewPolykeyServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Call HealthCheck
	r, err := c.HealthCheck(ctx, &emptypb.Empty{})
	if err != nil {
		log.Fatalf("could not health check: %v", err)
	}
	log.Printf("HealthCheck Response: Status=%s, Version=%s", r.GetStatus().String(), r.GetServiceVersion())

	// Call GetKey (example)
	getKeyReq := &pb.GetKeyRequest{
		KeyId: "test_key_123",
		RequesterContext: &pb.RequesterContext{ClientIdentity: "dev_client"},
	}
	getKeyResp, err := c.GetKey(ctx, getKeyReq)
	if err != nil {
		log.Fatalf("could not get key: %v", err)
	}
	log.Printf("GetKey Response: KeyId=%s, KeyType=%s", getKeyResp.GetMetadata().GetKeyId(), getKeyResp.GetMetadata().GetKeyType().String())

	// Call CreateKey (example)
	createKeyReq := &pb.CreateKeyRequest{
		KeyType: pb.KeyType_KEY_TYPE_AES_256,
		RequesterContext: &pb.RequesterContext{ClientIdentity: "dev_client"},
		Description: "My new test key",
		Tags: map[string]string{"environment": "development"},
	}
	createKeyResp, err := c.CreateKey(ctx, createKeyReq)
	if err != nil {
		log.Fatalf("could not create key: %v", err)
	}
	log.Printf("CreateKey Response: KeyId=%s, KeyType=%s", createKeyResp.GetMetadata().GetKeyId(), createKeyResp.GetMetadata().GetKeyType().String())
}
