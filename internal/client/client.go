package client

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

func main() {
	port := os.Getenv("POLYKEY_GRPC_PORT")
	if port == "" {
		port = "50051"
	}

	conn, err := grpc.NewClient(
		"localhost:"+port,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := pk.NewPolykeyServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := c.HealthCheck(ctx, &emptypb.Empty{})
	if err != nil {
		log.Fatalf("could not get health check: %v", err)
	}
	fmt.Printf("Health Check Response: %v\n", res)
}
