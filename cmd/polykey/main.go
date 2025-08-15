package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/spounge-ai/polykey/internal/app/grpc"
	app_errors "github.com/spounge-ai/polykey/internal/errors"
	"github.com/spounge-ai/polykey/internal/infra/audit"
	"github.com/spounge-ai/polykey/internal/infra/auth"
	infra_config "github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/service"
	"github.com/spounge-ai/polykey/internal/wiring"
)

const defaultTokenTTL = 1 * time.Hour

func main() {
	cfg, err := infra_config.Load(os.Getenv("POLYKEY_CONFIG_PATH"))
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	container := wiring.NewContainer(cfg, slog.Default())
	defer func() {
		if err := container.Close(); err != nil {
			slog.Error("failed to close container", "error", err)
		}
	}()

	deps, err := container.GetDependencies(context.Background())
	if err != nil {
		log.Fatalf("failed to get dependencies: %v", err)
	}

	logger := slog.Default()
	errorClassifier := app_errors.NewErrorClassifier(logger)

	keyService := service.NewKeyService(cfg, deps.KeyRepo, deps.KMSProviders, logger, errorClassifier)
	authService := service.NewAuthService(deps.ClientStore, deps.TokenManager, defaultTokenTTL)
	authorizer := auth.NewAuthorizer(cfg.Authorization, deps.KeyRepo)
	auditLogger := audit.NewAuditLogger(logger, deps.AuditRepo)

	srv, port, err := grpc.New(cfg, keyService, authService, authorizer, auditLogger, logger, errorClassifier)
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