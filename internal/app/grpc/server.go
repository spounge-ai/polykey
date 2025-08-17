package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"

	"github.com/spounge-ai/polykey/internal/app/grpc/interceptors"
	"github.com/spounge-ai/polykey/internal/domain"
	app_errors "github.com/spounge-ai/polykey/internal/errors"
	"github.com/spounge-ai/polykey/internal/infra/auth"
	"github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/infra/ratelimit"
	"github.com/spounge-ai/polykey/internal/service"
	"github.com/spounge-ai/polykey/pkg/patterns/lifecycle"
	"golang.org/x/time/rate"
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
	errorClassifier *app_errors.ErrorClassifier,
	tlsConfig *tls.Config,
) (*Server, int, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to listen: %w", err)
	}

	port := lis.Addr().(*net.TCPAddr).Port

	var opts []grpc.ServerOption
	if tlsConfig != nil {
		opts = append(opts, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}

	tokenStore := auth.NewInMemoryTokenStore()
	tokenManager, err := auth.NewTokenManager(cfg.BootstrapSecrets.JWTRSAPrivateKey, tokenStore, auditLogger)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create token manager for interceptor: %w", err)
	}

	// Create a rate limiter for the authentication interceptor.
	// Allow 10 requests per second with a burst of 20.
	rateLimiter := ratelimit.NewInMemoryRateLimiter(rate.Limit(10), 20)

	opts = append(opts, grpc.ChainUnaryInterceptor(
		interceptors.UnaryLoggingInterceptor(logger),
		interceptors.AuthenticationInterceptor(tokenManager, rateLimiter),
		interceptors.UnaryValidationInterceptor(errorClassifier),
	))

	grpcServer := grpc.NewServer(opts...)

	deps := PolykeyDeps{
		Config:          cfg,
		KeyService:      keyService,
		AuthService:     authService,
		Authorizer:      authorizer,
		Audit:           auditLogger,
		Logger:          logger,
		ErrorClassifier: errorClassifier,
	}

	polykeyService := NewPolykeyService(deps)
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

func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("gRPC server listening", "address", s.lis.Addr().String())
	s.healthSrv.SetServingStatus("polykey.v2.PolykeyService", grpc_health_v1.HealthCheckResponse_SERVING)
	return s.grpcServer.Serve(s.lis)
}

func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping gRPC server...")
	s.healthSrv.SetServingStatus("polykey.v2.PolykeyService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	s.grpcServer.GracefulStop()
	if err := s.lis.Close(); err != nil {
		s.logger.Error("failed to close listener", "error", err)
		return err
	}
	s.logger.Info("gRPC server stopped.")
	return nil
}

func (s *Server) Health(ctx context.Context) lifecycle.HealthStatus {
	// The gRPC health server manages the status internally, so we just reflect that.
	// A more sophisticated check could involve checking dependencies.
	return lifecycle.HealthStatus{Ready: true, Message: "gRPC server is running"}
}
