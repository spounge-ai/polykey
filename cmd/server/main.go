package main

import (
	"fmt"
	"log"
	"net"
	"os"

	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"github.com/spounge-ai/polykey/internal/service"
	"google.golang.org/grpc"
)

func main() {
	port := os.Getenv("POLYKEY_GRPC_PORT")
	if port == "" {
		port = "50051"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pk.RegisterPolykeyServiceServer(s, &service.MockPolykeyService{})

	fmt.Printf("Polykey gRPC server listening on :%s\n", port)
	if err := s.Serve(lis);
	err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
