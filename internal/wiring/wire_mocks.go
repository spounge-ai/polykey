//go:build local_mocks

package wiring

import (
	"fmt"

	"github.com/spounge-ai/polykey/internal/domain"
	infra_config "github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/tests/mocks/kms"
	"github.com/spounge-ai/polykey/tests/mocks/persistence"
)

// ProvideDependencies builds and returns the mock services for testing.
func ProvideDependencies(cfg *infra_config.Config) (domain.KMSService, domain.KeyRepository, error) {
	if cfg.AWS.Enabled {
		return nil, nil, fmt.Errorf("AWS is enabled in the configuration, but the mock build tag is provided")
	}

	kmsAdapter := kms.NewMockKMSAdapter()
	keyRepo := persistence.NewMockS3Storage()

	return kmsAdapter, keyRepo, nil
}
