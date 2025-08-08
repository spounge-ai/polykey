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
		port = "50053" // Default port
	}

	conn, err := grpc.NewClient("localhost:" + port, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("failed to close connection: %v", err)
		}
	}()

	c := pb.NewPolykeyServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Call HealthCheck
	r, err := c.HealthCheck(ctx, &emptypb.Empty{})
	if err != nil {
		log.Fatalf("could not health check: %v", err)
	}
	log.Printf("HealthCheck Response: Status=%s, Version=%s", r.GetStatus().String(), r.GetServiceVersion())

	// Call CreateKey to generate a new key first
	createKeyReq := &pb.CreateKeyRequest{
		KeyType:          pb.KeyType_KEY_TYPE_AES_256,
		RequesterContext: &pb.RequesterContext{ClientIdentity: "admin"}, // Run as admin to pass authorization
		Description:      "A key created by the dev client",
		Tags:             map[string]string{"source": "dev_client"},
	}
	createKeyResp, err := c.CreateKey(ctx, createKeyReq)
	if err != nil {
		log.Fatalf("could not create key: %v", err)
	}
	log.Printf("CreateKey Response: KeyId=%s, KeyType=%s", createKeyResp.GetMetadata().GetKeyId(), createKeyResp.GetMetadata().GetKeyType().String())

 	newKeyId := createKeyResp.GetMetadata().GetKeyId()
	log.Printf("Attempting to get the key we just created: %s", newKeyId)
	getKeyReq := &pb.GetKeyRequest{
		KeyId:            newKeyId,
		RequesterContext: &pb.RequesterContext{ClientIdentity: "dev_client"},
	}
	getKeyResp, err := c.GetKey(ctx, getKeyReq)
	if err != nil {
		log.Fatalf("could not get key: %v", err)
	}
	log.Printf("GetKey Response: Successfully retrieved key %s", getKeyResp.GetMetadata().GetKeyId())
}
