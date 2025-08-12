//go:build local_mocks

package wiring

import (
	"fmt"

	"github.com/spounge-ai/polykey/internal/domain"
	infra_config "github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/kms"
	kms_mocks "github.com/spounge-ai/polykey/tests/mocks/kms"
	"github.com/spounge-ai/polykey/tests/mocks/persistence"
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
	if cfg.AWS.Enabled {
		return nil, fmt.Errorf("AWS is enabled in the configuration, but the mock build tag is provided")
	}

	providers := make(map[string]kms.KMSProvider)
	providers["local"] = kms_mocks.NewMockKMSAdapter()
	return providers, nil
}

func provideKeyRepository(cfg *infra_config.Config) (domain.KeyRepository, error) {
	return persistence.NewInMemoryKeyRepository(), nil
}
