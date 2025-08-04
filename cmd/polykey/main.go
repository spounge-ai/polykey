package main

import (
	"log"
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
	auditLogger := audit.NewAuditLogger()

	srv, _, err := grpc.New(cfg, keyRepo, kmsAdapter, authorizer, auditLogger)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	if err := srv.RunBlocking(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
