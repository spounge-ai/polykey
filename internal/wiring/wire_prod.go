//go:build !local_mocks

package wiring

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spounge-ai/polykey/internal/domain"
	infra_config "github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/infra/persistence"
	"github.com/spounge-ai/polykey/internal/kms"
)

var (
	pgxPoolOnce sync.Once
	pgxPool     *pgxpool.Pool
)

func providePgxPool(cfg *infra_config.Config) (*pgxpool.Pool, error) {
	var err error
	pgxPoolOnce.Do(func() {
		pgxPool, err = pgxpool.New(context.Background(), cfg.NeonDB.URL)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create new pgxpool: %w", err)
	}
	return pgxPool, nil
}

func ProvideDependencies(cfg *infra_config.Config) (map[string]kms.KMSProvider, domain.KeyRepository, domain.AuditRepository, error) {
	kmsProviders, err := provideKMSProviders(cfg)
	if err != nil {
		return nil, nil, nil, err
	}

	dbpool, err := providePgxPool(cfg)
	if err != nil {
		return nil, nil, nil, err
	}

	keyRepo, err := provideKeyRepository(dbpool)
	if err != nil {
		return nil, nil, nil, err
	}

	auditRepo, err := provideAuditRepository(dbpool)
	if err != nil {
		return nil, nil, nil, err
	}

	return kmsProviders, keyRepo, auditRepo, nil
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

func provideKeyRepository(dbpool *pgxpool.Pool) (domain.KeyRepository, error) {
	return persistence.NewNeonDBStorage(dbpool)
}

//nolint:unused
func provideS3Storage(cfg *infra_config.Config) (domain.KeyRepository, error) {
	awsCfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(cfg.AWS.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config: %w", err)
	}
	return persistence.NewS3Storage(awsCfg, cfg.AWS.S3Bucket, slog.Default())
}

func provideAuditRepository(dbpool *pgxpool.Pool) (domain.AuditRepository, error) {
	return persistence.NewAuditRepository(dbpool)
}
