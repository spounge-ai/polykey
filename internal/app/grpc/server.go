package grpc

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/spounge-ai/polykey/internal/app/grpc/interceptors"
	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/internal/infra/auth"
	"github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

type Server struct {
	grpcServer *grpc.Server
	healthSrv  *health.Server
	cfg        *config.Config
	lis        net.Listener
	logger     *slog.Logger
}

func New(
	cfg *config.Config,
	keyService service.KeyService,
	authService service.AuthService,
	authorizer domain.Authorizer,
	auditLogger domain.AuditLogger,
	logger *slog.Logger,
) (*Server, int, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to listen: %w", err)
	}

	port := lis.Addr().(*net.TCPAddr).Port

	var opts []grpc.ServerOption
	if cfg.Server.TLS.Enabled {
		creds, err := credentials.NewServerTLSFromFile(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to load TLS credentials: %w", err)
		}
		opts = append(opts, grpc.Creds(creds))
	}

	// This part is tricky because the TokenManager is needed for the interceptor,
	// but it's created deep within the wiring. For now, we might need to create it here
	// or pass it up from the wiring.
	// Let's assume for now it's passed up or created here from config.
	tokenManager, err := auth.NewTokenManager(cfg.BootstrapSecrets.JWTRSAPrivateKey)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create token manager for interceptor: %w", err)
	}

	opts = append(opts, grpc.ChainUnaryInterceptor(
		interceptors.UnaryLoggingInterceptor(),
		interceptors.AuthenticationInterceptor(tokenManager),
	))

	grpcServer := grpc.NewServer(opts...)

	polykeyService, err := NewPolykeyService(cfg, keyService, authService, authorizer, auditLogger, logger)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create polykey service: %w", err)
	}

	pk.RegisterPolykeyServiceServer(grpcServer, polykeyService)

	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSrv)
	reflection.Register(grpcServer)

	return &Server{
		grpcServer: grpcServer,
		healthSrv:  healthSrv,
		cfg:        cfg,
		lis:        lis,
		logger:     logger,
	}, port, nil
}

func (s *Server) Start() error {
	s.logger.Info("gRPC server listening", "address", s.lis.Addr().String())
	s.healthSrv.SetServingStatus("polykey.v2.PolykeyService", grpc_health_v1.HealthCheckResponse_SERVING)
	return s.grpcServer.Serve(s.lis)
}

func (s *Server) Stop() {
	s.logger.Info("Stopping gRPC server...")
	s.healthSrv.SetServingStatus("polykey.v2.PolykeyService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	s.grpcServer.GracefulStop()
	if err := s.lis.Close(); err != nil {
		s.logger.Error("failed to close listener", "error", err)
	}
	s.logger.Info("gRPC server stopped.")
}