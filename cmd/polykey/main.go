package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spounge-ai/polykey/internal/app/grpc"
	app_errors "github.com/spounge-ai/polykey/internal/errors"
	"github.com/spounge-ai/polykey/internal/infra/audit"
	"github.com/spounge-ai/polykey/internal/infra/auth"
	infra_config "github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/service"
	"github.com/spounge-ai/polykey/internal/wiring"
	"github.com/spounge-ai/polykey/pkg/patterns/lifecycle"
)

const defaultTokenTTL = 1 * time.Hour

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg, err := infra_config.Load(os.Getenv("POLYKEY_CONFIG_PATH"))
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	tlsConfig, err := wiring.ConfigureTLS(cfg.Server.TLS)
	if err != nil {
		logger.Error("failed to configure TLS", "error", err)
		os.Exit(1)
	}

	container := wiring.NewContainer(cfg, logger)
	defer func() {
		if err := container.Close(); err != nil {
			logger.Error("failed to close container", "error", err)
		}
	}()

	deps, err := container.GetDependencies(ctx)
	if err != nil {
		logger.Error("failed to get dependencies", "error", err)
		os.Exit(1)
	}

	errorClassifier := app_errors.NewErrorClassifier(logger)
	auditLogger := audit.NewAuditLogger(logger, deps.AuditRepo)
	authorizer := auth.NewAuthorizer(cfg.Authorization, deps.KeyRepo, auditLogger)
	keyService := service.NewKeyService(cfg, deps.KeyRepo, deps.KMSProviders, logger, errorClassifier, auditLogger)
	authService := service.NewAuthService(deps.ClientStore, deps.TokenManager, defaultTokenTTL)

	srv, port, err := grpc.New(cfg, keyService, authService, authorizer, auditLogger, logger, errorClassifier, tlsConfig)
	if err != nil {
		logger.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	// Set up resource management
	resourceManager := []lifecycle.ManagedResource{srv}

	// Start resources in a separate goroutine
	go func() {
		logger.Info("starting application resources")
		for _, r := range resourceManager {
			if err := r.Start(ctx); err != nil {
				logger.Error("error starting resource", "error", err)
				cancel() // Trigger shutdown
				return
			}
		}
		logger.Info("application started successfully", "port", port)
	}()

	// Wait for shutdown signal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case s := <-signalChan:
		logger.Info("received shutdown signal", "signal", s.String())
	case <-ctx.Done():
		logger.Info("context cancelled, initiating shutdown")
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	logger.Info("shutting down application resources")
	for i := len(resourceManager) - 1; i >= 0; i-- {
		if err := resourceManager[i].Stop(shutdownCtx); err != nil {
			logger.Error("error stopping resource", "error", err)
		}
	}
	logger.Info("shutdown complete")
}
