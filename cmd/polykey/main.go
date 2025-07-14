package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"github.com/spounge-ai/polykey-service/internal/server"
	"github.com/spounge-ai/polykey-service/internal/service"

	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v1"
)

func main() {
	// Listen on TCP port 50051
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Create new gRPC server
	grpcServer := grpc.NewServer()

	// Initialize service (replace with your real implementation if needed)
	svc := service.NewMockService()

	// Register PolykeyService server using new proto package 'pk'
	pk.RegisterPolykeyServiceServer(grpcServer, server.NewServer(svc))

	log.Println("Polykey server running on port 50051")

	// Channel to listen for interrupt or terminate signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Run gRPC server in a goroutine
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// Wait here until an OS signal is received
	<-stop

	log.Println("Shutting down Polykey server...")

	grpcServer.GracefulStop()
	if err := lis.Close(); err != nil {
		log.Printf("Error closing listener: %v", err)
	}

	log.Println("Polykey server terminated")
}
