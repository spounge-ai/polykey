//go:build !local_mocks

package wiring

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spounge-ai/polykey/internal/domain"
	infra_config "github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/kms"
	"github.com/spounge-ai/polykey/internal/infra/persistence"
)

func ProvideDependencies(cfg *infra_config.Config) (map[string]kms.KMSProvider, domain.KeyRepository, error) {
	kmsProviders, err := provideKMSProviders(cfg)
	if err != nil {
		return nil, nil, err
	}

	keyRepo, err := provideKeyRepository(cfg)
	if err != nil {
		return nil, nil, err
	}

	return kmsProviders, keyRepo, nil
}

func provideKMSProviders(cfg *infra_config.Config) (map[string]kms.KMSProvider, error) {
	providers := make(map[string]kms.KMSProvider)

	if cfg.LocalMasterKey != "" {
		localProvider, err := kms.NewLocalKMSProvider(cfg.LocalMasterKey)
		if err != nil {
			return nil, err
		}
		providers["local"] = localProvider
	}

	if cfg.AWS.Enabled {
		awsCfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(cfg.AWS.Region))
		if err != nil {
			return nil, fmt.Errorf("failed to load aws config: %w", err)
		}
		awsProvider := kms.NewAWSKMSProvider(awsCfg, cfg.AWS.KMSKeyARN)
		providers["aws"] = awsProvider
	}

	return providers, nil
}

func provideKeyRepository(cfg *infra_config.Config) (domain.KeyRepository, error) {
	switch cfg.Persistence.Type {
	case "s3":
		return provideS3Storage(cfg)
	case "neondb":
		return provideNeonDBStorage(cfg)
	case "cockroachdb":
		return provideCockroachDBStorage(cfg)
	default:
		return nil, fmt.Errorf("invalid persistence type: %s", cfg.Persistence.Type)
	}
}

func provideS3Storage(cfg *infra_config.Config) (domain.KeyRepository, error) {
	awsCfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(cfg.AWS.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config: %w", err)
	}
	return persistence.NewS3Storage(awsCfg, cfg.AWS.S3Bucket, slog.Default())
}

func provideNeonDBStorage(cfg *infra_config.Config) (domain.KeyRepository, error) {
	dbpool, err := pgxpool.New(context.Background(), cfg.NeonDB.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to create new pgxpool: %w", err)
	}
	return persistence.NewNeonDBStorage(dbpool)
}

func provideCockroachDBStorage(cfg *infra_config.Config) (domain.KeyRepository, error) {
	dbpool, err := pgxpool.New(context.Background(), cfg.CockroachDB.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to create new pgxpool: %w", err)
	}
	return persistence.NewCockroachDBStorage(dbpool)
}