package main

import (
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/spounge-ai/polykey/internal/app/grpc"
	"github.com/spounge-ai/polykey/internal/infra/audit"
	"github.com/spounge-ai/polykey/internal/infra/auth"
	infra_config "github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/service"
	"github.com/spounge-ai/polykey/internal/validation"
	"github.com/spounge-ai/polykey/internal/wiring"
	"github.com/spounge-ai/polykey/pkg/errors"
)

const (
	defaultTokenTTL = 1 * time.Hour
)

func main() {
	cfg, err := infra_config.Load(os.Getenv("POLYKEY_CONFIG_PATH"))
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	kmsProviders, keyRepo, auditRepo, clientStore, tokenManager, err := wiring.ProvideDependencies(cfg)
	if err != nil {
		log.Fatalf("failed to provide dependencies: %v", err)
	}

	logger := slog.Default()

	keyService := service.NewKeyService(cfg, keyRepo, kmsProviders, logger)
	authService := service.NewAuthService(clientStore, tokenManager, defaultTokenTTL)
	authorizer := auth.NewAuthorizer(cfg.Authorization, keyRepo)
	auditLogger := audit.NewAuditLogger(logger, auditRepo)
	errorClassifier := errors.NewErrorClassifier(logger)
	validator, err := validation.NewRequestValidator()
	if err != nil {
		log.Fatalf("failed to create request validator: %v", err)
	}
	queryValidator := validation.NewQueryValidator()

	srv, port, err := grpc.New(cfg, keyService, authService, authorizer, auditLogger, logger, errorClassifier, validator, queryValidator)
	if err != nil {
		slog.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	slog.Info("starting gRPC server", "port", port)
	if err := srv.Start(); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}