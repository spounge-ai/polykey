package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/spounge-ai/polykey/internal/config"
	"github.com/spounge-ai/polykey/internal/service"
	"github.com/spounge-ai/polykey/internal/storage"
	pb "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// Server represents the gRPC server.
type Server struct {
	grpcServer *grpc.Server
	healthSrv  *health.Server
	cfg        *config.Config
	lis        net.Listener
}

// New creates a new gRPC server.
func New(cfg *config.Config) (*Server, int, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to listen: %w", err)
	}

	// Get the dynamically assigned port
	port := lis.Addr().(*net.TCPAddr).Port

	var opts []grpc.ServerOption
	if cfg.Server.TLS.Enabled {
		creds, err := credentials.NewServerTLSFromFile(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to load TLS credentials: %w", err)
		}
		opts = append(opts, grpc.Creds(creds))
	}

	grpcServer := grpc.NewServer(opts...)

	// Create a new Polykey service
	storageService, err := storage.NewVaultStorage(cfg.Vault.Address, cfg.Vault.Token)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create vault storage: %w", err)
	}
	polykeyService, err := service.NewPolykeyService(cfg, storageService)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create polykey service: %w", err)
	}

	// Register the Polykey service
	pb.RegisterPolykeyServiceServer(grpcServer, polykeyService)

	// Debug: Print registered services
	log.Printf("Registering services...")
	for name := range grpcServer.GetServiceInfo() {
		log.Printf("Registered service: %s", name)
	}

	// Register health and reflection services
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSrv)
	reflection.Register(grpcServer)

	// Print all services again after health/reflection
	log.Printf("All registered services:")
	for name := range grpcServer.GetServiceInfo() {
		log.Printf("  - %s", name)
	}

	return &Server{
		grpcServer: grpcServer,
		healthSrv:  healthSrv,
		cfg:        cfg,
		lis:        lis,
	}, port, nil
}

// Start starts the gRPC server.
func (s *Server) Start() error {
	log.Printf("gRPC server listening on %s", s.lis.Addr().String())
	s.healthSrv.SetServingStatus("polykey.v2.PolykeyService", grpc_health_v1.HealthCheckResponse_SERVING)
	return s.grpcServer.Serve(s.lis)
}

// Stop gracefully stops the gRPC server.
func (s *Server) Stop() {
	log.Println("Stopping gRPC server...")
	s.healthSrv.SetServingStatus("polykey.v2.PolykeyService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	s.grpcServer.GracefulStop()
	log.Println("gRPC server stopped.")
}

// MustNew is like New but panics on error.
func MustNew() *Server {
	cfg, err := config.Load(os.Getenv("POLYKEY_CONFIG_PATH"))
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	srv, _, err := New(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create server: %v", err))
	}
	return srv
}

// Run starts the server and waits for a signal to stop it.
func (s *Server) Run(ctx context.Context) error {
	// Create a channel to listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		log.Printf("Starting gRPC server on %s", s.lis.Addr())
		if err := s.Start(); err != nil {
			errChan <- fmt.Errorf("failed to start gRPC server: %w", err)
		}
	}()

	// Wait for either an error, context cancellation, or interrupt signal
	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		log.Println("Context cancelled, shutting down server...")
		s.Stop()
		return ctx.Err()
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down server...", sig)
		s.Stop()
		return nil
	}
}

// RunBlocking starts the server and blocks until interrupted
func (s *Server) RunBlocking() error {
	// Create a channel to listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		log.Printf("Starting gRPC server on %s", s.lis.Addr())
		if err := s.Start(); err != nil {
			errChan <- fmt.Errorf("failed to start gRPC server: %w", err)
		}
	}()

	// Wait for either an error or interrupt signal
	select {
	case err := <-errChan:
		return err
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down server...", sig)
		s.Stop()
		return nil
	}
}

// WithTimeout adds a timeout to a context.
func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}