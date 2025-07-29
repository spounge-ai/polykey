package client

import (
	"context"
	"fmt"
	"os"
	"time"

	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

func RunClient() error {
	port := os.Getenv("POLYKEY_GRPC_PORT")
	if port == "" {
		port = "50051"
	}

	conn, err := grpc.NewClient(
		"localhost:"+port,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("did not connect: %w", err)
	}
	defer conn.Close()

	c := pk.NewPolykeyServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	fmt.Printf("Client attempting to call service: %s\n", pk.PolykeyService_HealthCheck_FullMethodName)

	res, err := c.HealthCheck(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not get health check: %w", err)
	}
	fmt.Printf("Health Check Response: %v\n", res)
	return nil
}
