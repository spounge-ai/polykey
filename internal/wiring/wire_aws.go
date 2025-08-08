//go:build !local_mocks

package wiring

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/spounge-ai/polykey/internal/domain"
	infra_aws "github.com/spounge-ai/polykey/internal/infra/aws"
	infra_config "github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/infra/kms"
	"github.com/spounge-ai/polykey/internal/infra/persistence"
)

func ProvideDependencies(cfg *infra_config.Config) (domain.KMSService, domain.KeyRepository, error) {
	var kmsService domain.KMSService
	var err error

	awsCfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(cfg.AWS.Region))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load aws config: %w", err)
	}

	if cfg.LocalMasterKey != "" {
		kmsService, err = kms.NewLocalKMSService(cfg.LocalMasterKey)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create local kms service: %w", err)
		}
	} else if cfg.AWS.Enabled {
		cacheTTL, err := time.ParseDuration(cfg.AWS.CacheTTL)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse cache TTL: %w", err)
		}

		kmsService = infra_aws.NewKMSCachedAdapter(infra_aws.NewKMSAdapter(awsCfg, cfg.AWS.KMSKeyARN), cacheTTL)
	} else {
		return nil, nil, fmt.Errorf("no kms service configured")
	}

	var keyRepo domain.KeyRepository
	switch cfg.Persistence.Type {
	case "s3":
		keyRepo, err = persistence.NewS3Storage(awsCfg, cfg.AWS.S3Bucket, slog.Default())
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create s3 storage: %w", err)
		}
	case "neondb":
		keyRepo, err = persistence.NewNeonDBStorage()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create neondb storage: %w", err)
		}
	default:
		return nil, nil, fmt.Errorf("invalid persistence type: %s", cfg.Persistence.Type)
	}

	return kmsService, keyRepo, nil
}
