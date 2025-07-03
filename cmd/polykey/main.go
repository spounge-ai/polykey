package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"github.com/SpoungeAI/polykey-service/internal/server"
	"github.com/SpoungeAI/polykey-service/internal/service" 
	pb "github.com/SpoungeAI/polykey-service/pkg/polykey/pb"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	svc := service.NewMockService()
	pb.RegisterPolyKeyServer(grpcServer, server.NewServer(svc))

	log.Println("PolyKey server running on port 50051")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	<-stop

	log.Println("Shutting down PolyKey server...")

	grpcServer.Stop()
	lis.Close()

	log.Println("PolyKey server terminated")
}
