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
	"github.com/spounge-ai/polykey/internal/infra/persistence"
)

// ProvideDependencies builds and returns the real, production-ready services.
func ProvideDependencies(cfg *infra_config.Config) (domain.KMSService, domain.KeyRepository, error) {
	if !cfg.AWS.Enabled {
		return nil, nil, fmt.Errorf("AWS is not enabled in the configuration, but no mock build tag was provided")
	}

	awsCfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(cfg.AWS.Region))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load aws config: %w", err)
	}

	cacheTTL, err := time.ParseDuration(cfg.AWS.CacheTTL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse cache TTL: %w", err)
	}

	kmsAdapter := infra_aws.NewKMSCachedAdapter(infra_aws.NewKMSAdapter(awsCfg), cacheTTL)
	keyRepo, err := persistence.NewS3Storage(awsCfg, cfg.AWS.S3Bucket, slog.Default())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create s3 storage: %w", err)
	}

	return kmsAdapter, keyRepo, nil
}
