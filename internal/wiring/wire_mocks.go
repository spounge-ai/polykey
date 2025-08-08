//go:build local_mocks

package wiring

import (
	"fmt"

	"github.com/spounge-ai/polykey/internal/domain"
	infra_config "github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/tests/mocks/kms"
	"github.com/spounge-ai/polykey/tests/mocks/persistence"
)

func ProvideDependencies(cfg *infra_config.Config) (domain.KMSService, domain.KeyRepository, error) {
	kmsService, err := provideKMSService(cfg)
	if err != nil {
		return nil, nil, err
	}

	keyRepo, err := provideKeyRepository(cfg)
	if err != nil {
		return nil, nil, err
	}

	return kmsService, keyRepo, nil
}

func provideKMSService(cfg *infra_config.Config) (domain.KMSService, error) {
	if cfg.AWS.Enabled {
		return nil, fmt.Errorf("AWS is enabled in the configuration, but the mock build tag is provided")
	}

	return kms.NewMockKMSAdapter(), nil
}

func provideKeyRepository(cfg *infra_config.Config) (domain.KeyRepository, error) {
	return persistence.NewMockS3Storage(), nil
}