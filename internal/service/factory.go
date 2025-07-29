package service

import (
	"github.com/spounge-ai/polykey/internal/audit"
	"github.com/spounge-ai/polykey/internal/authz"
	"github.com/spounge-ai/polykey/internal/config"
	"github.com/spounge-ai/polykey/internal/keymanager"
	"github.com/spounge-ai/polykey/internal/storage"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// ServiceFactory defines the interface for creating a PolykeyService.
type ServiceFactory interface {
	Create(cfg *config.Config, s storage.Storage) (pk.PolykeyServiceServer, error)
}

// NewServiceFactory creates a new instance of ServiceFactory.
func NewServiceFactory() ServiceFactory {
	return &serviceFactoryImpl{}
}

// serviceFactoryImpl implements the ServiceFactory interface.
type serviceFactoryImpl struct{}

// Create creates a new instance of PolykeyService.
func (f *serviceFactoryImpl) Create(cfg *config.Config, s storage.Storage) (pk.PolykeyServiceServer, error) {
	return NewPolykeyService(cfg, s, authz.NewAuthorizer(), audit.NewLogger(), keymanager.NewKeyManager())
}
