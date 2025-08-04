package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/spounge-ai/polykey/internal/app/grpc"
	"github.com/spounge-ai/polykey/internal/infra/audit"
	"github.com/spounge-ai/polykey/internal/infra/auth"
	infra_config "github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/wiring"
)

func main() {
	cfg, err := infra_config.Load(os.Getenv("POLYKEY_CONFIG_PATH"))
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	kmsAdapter, keyRepo, err := wiring.ProvideDependencies(cfg)
	if err != nil {
		log.Fatalf("failed to provide dependencies: %v", err)
	}

	authorizer := auth.NewAuthorizer()
	auditLogger := audit.NewAuditLogger(slog.Default(), nil) // nil for audit repo for now

	srv, port, err := grpc.New(cfg, keyRepo, kmsAdapter, authorizer, auditLogger, slog.Default())
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
