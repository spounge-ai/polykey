package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/spounge-ai/polykey/internal/app/grpc"
	"github.com/spounge-ai/polykey/internal/domain"
	infra_auth "github.com/spounge-ai/polykey/internal/infra/auth"
	infra_aws "github.com/spounge-ai/polykey/internal/infra/aws"
	infra_config "github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/infra/persistence"

	dev_auth "github.com/spounge-ai/polykey/dev/auth"
	dev_persistence "github.com/spounge-ai/polykey/dev/persistence"
)

func main() {
	cfg, err := infra_config.Load(os.Getenv("POLYKEY_CONFIG_PATH"))
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	var kmsAdapter domain.KMSService
	var authorizer domain.Authorizer
	var keyRepo domain.KeyRepository

	env := os.Getenv("POLYKEY_ENV")
	if env == "dev" || env == "test" {
		log.Println("Running in DEV/TEST environment: Using mock implementations.")
		kmsAdapter = &mockKMSAdapter{} // Assuming a mock KMS adapter exists or will be created
		authorizer = dev_auth.NewMockAuthorizer()
		keyRepo = dev_persistence.NewMockVaultStorage()
	} else {
		log.Println("Running in PROD environment: Using real implementations.")
		awsCfg, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			log.Fatalf("failed to load aws config: %v", err)
		}
		kmsAdapter = infra_aws.NewKMSCachedAdapter(infra_aws.NewKMSAdapter(awsCfg), 5*time.Minute)
		authorizer = infra_auth.NewAuthorizer()
		keyRepo, err = persistence.NewVaultStorage(cfg.Vault.Address, cfg.Vault.Token)
		if err != nil {
			log.Fatalf("failed to create vault storage: %v", err)
		}
	}

	srv, _, err := grpc.New(cfg, keyRepo, kmsAdapter, authorizer, nil) // nil for audit logger for now
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	if err := srv.RunBlocking(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// mockKMSAdapter is a placeholder for a mock KMS service.
type mockKMSAdapter struct{}

func (m *mockKMSAdapter) EncryptDEK(ctx context.Context, plaintextDEK []byte, masterKeyID string) ([]byte, error) {
	return []byte("mock_encrypted_dek"), nil
}

func (m *mockKMSAdapter) DecryptDEK(ctx context.Context, encryptedDEK []byte, masterKeyID string) ([]byte, error) {
	return []byte("mock_plaintext_dek"), nil
}
