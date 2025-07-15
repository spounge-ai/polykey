package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	// NEW: 1. Import the required health check packages
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

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

	// NEW: 2. Create an instance of the health server
	healthServer := health.NewServer()

	// Initialize service
	svc := service.NewMockService()

	// Register PolykeyService server
	pk.RegisterPolykeyServiceServer(grpcServer, server.NewServer(svc))

	// NEW: 3. Register the health service on the same gRPC server
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	// NEW: 4. Set the initial serving status for your service.
	// The service name is defined in your .proto file as `package.Service`.
	healthServer.SetServingStatus("polykey.v1.PolykeyService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING) // For overall server health

	log.Println("Polykey server with health check running on port 50051")

	// Channel to listen for interrupt or terminate signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Run gRPC server in a goroutine
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server stopped: %v", err)
		}
	}()

	// Wait here until an OS signal is received
	<-stop
	log.Println("Shutting down Polykey server...")

	// Set all services to NOT_SERVING status before shutdown
	healthServer.Shutdown()

	grpcServer.GracefulStop()
	if err := lis.Close(); err != nil {
		log.Printf("Error closing listener: %v", err)
	}

	log.Println("Polykey server terminated")
}