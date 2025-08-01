package main

import (
	"log"
	"os"

	"github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/app/grpc"
)

func main() {
	cfg, err := infra_config.Load(os.Getenv("POLYKEY_CONFIG_PATH"))
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	awsCfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("failed to load aws config: %v", err)
	}

	kmsAdapter := infra_aws.NewKMSCachedAdapter(infra_aws.NewKMSAdapter(awsCfg))
	authorizer := infra_auth.NewAuthorizer()
	keyRepo, err := persistence.NewVaultStorage(cfg.Vault.Address, cfg.Vault.Token)
	if err != nil {
		log.Fatalf("failed to create vault storage: %v", err)
	}

	srv, _, err := grpc.New(cfg, keyRepo, kmsAdapter, authorizer, nil) // nil for audit logger for now
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	if err := srv.RunBlocking(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}