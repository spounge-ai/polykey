package wiring

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spounge-ai/polykey/internal/domain"
	infra_audit "github.com/spounge-ai/polykey/internal/infra/audit"
	infra_auth "github.com/spounge-ai/polykey/internal/infra/auth"
	infra_config "github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/infra/persistence"
	"github.com/spounge-ai/polykey/internal/kms"
)

type Container struct {
	config       *infra_config.Config
	logger       *slog.Logger
	pgxPool      *pgxpool.Pool
	pgxPoolOnce  sync.Once
	kmsProviders map[string]kms.KMSProvider
	keyRepo      domain.KeyRepository
	auditRepo    domain.AuditRepository
	clientStore  domain.ClientStore
	tokenManager *infra_auth.TokenManager
	tokenStore   infra_auth.TokenStore
	auditLogger  domain.AuditLogger
}

func NewContainer(cfg *infra_config.Config, logger *slog.Logger) *Container {
	if logger == nil {
		logger = slog.Default()
	}
	return &Container{config: cfg, logger: logger}
}

type Dependencies struct {
	KMSProviders map[string]kms.KMSProvider
	KeyRepo      domain.KeyRepository
	AuditRepo    domain.AuditRepository
	ClientStore  domain.ClientStore
	TokenManager *infra_auth.TokenManager
}

func (c *Container) GetDependencies(ctx context.Context) (*Dependencies, error) {
	if err := c.initializeAll(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize dependencies: %w", err)
	}
	return &Dependencies{
		KMSProviders: c.kmsProviders,
		KeyRepo:      c.keyRepo,
		AuditRepo:    c.auditRepo,
		ClientStore:  c.clientStore,
		TokenManager: c.tokenManager,
	}, nil
}

func (c *Container) initializeAll(ctx context.Context) error {
	initializers := []func(context.Context) error{
		c.initPgxPool,
		c.initKMSProviders,
		func(context.Context) error { return c.initTokenStore() },
		func(context.Context) error { return c.initKeyRepository() },
		func(context.Context) error { return c.initAuditRepository() },
		func(context.Context) error { return c.initAuditLogger() },
		func(context.Context) error { return c.initClientStore() },
		func(context.Context) error { return c.initTokenManager() },
	}
	for _, initFn := range initializers {
		if err := initFn(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (c *Container) initAuditLogger() error {
	if c.auditLogger != nil {
		return nil
	}
	if c.auditRepo == nil {
		return fmt.Errorf("audit repository not initialized")
	}
	c.auditLogger = infra_audit.NewAuditLogger(c.logger, c.auditRepo)
	c.logger.Debug("initialized audit logger")
	return nil
}

func (c *Container) GetPgxPool(ctx context.Context) (*pgxpool.Pool, error) {
	if err := c.initPgxPool(ctx); err != nil {
		return nil, err
	}
	return c.pgxPool, nil
}

func (c *Container) initPgxPool(ctx context.Context) error {
	var err error
	c.pgxPoolOnce.Do(func() {
		dbConfig := infra_config.NeonDBConfig{URL: c.config.BootstrapSecrets.NeonDBURLDevelopment}
		c.pgxPool, err = persistence.NewSecureConnectionPool(ctx, dbConfig, c.config.Server, c.config.Persistence)
		if err != nil {
			c.logger.Error("failed to create database connection pool", "error", err)
		}
	})
	return err
}

func (c *Container) initKMSProviders(ctx context.Context) error {
	if c.kmsProviders != nil {
		return nil
	}
	c.kmsProviders = make(map[string]kms.KMSProvider)

	// Initialize local provider if configured
	if c.config.BootstrapSecrets.PolykeyMasterKey != "" {
		localProvider, err := kms.NewLocalKMSProvider(c.config.BootstrapSecrets.PolykeyMasterKey)
		if err != nil {
			return fmt.Errorf("failed to create local KMS provider: %w", err)
		}
		c.kmsProviders["local"] = localProvider
		c.logger.Debug("initialized local KMS provider")
	}

	// Initialize AWS provider if configured
	if c.config.AWS.Enabled {
		awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(c.config.AWS.Region))
		if err != nil {
			return fmt.Errorf("failed to load AWS config: %w", err)
		}
		c.kmsProviders["aws"] = kms.NewAWSKMSProvider(awsCfg, c.config.AWS.KMSKeyARN)
		c.logger.Debug("initialized AWS KMS provider", "region", c.config.AWS.Region)
	}

	if len(c.kmsProviders) == 0 {
		return fmt.Errorf("no KMS provider configured")
	}

	return nil
}

func (c *Container) initKeyRepository() error {
	if c.keyRepo != nil {
		return nil
	}
	if c.pgxPool == nil {
		return fmt.Errorf("database pool not initialized")
	}
	var err error
	// Create the base repository
	baseRepo, err := persistence.NewNeonDBStorage(c.pgxPool, c.logger)
	if err != nil {
		return err
	}

	// Check if the circuit breaker is enabled
	if c.config.Persistence.CircuitBreaker.Enabled {
		c.logger.Debug("wrapping key repository with circuit breaker")
		c.keyRepo = persistence.NewKeyRepositoryCircuitBreaker(
			baseRepo,
			c.config.Persistence.CircuitBreaker.MaxFailures,
			c.config.Persistence.CircuitBreaker.ResetTimeout,
		)
	} else {
		c.keyRepo = baseRepo
	}

	c.logger.Debug("initialized key repository")
	return nil
}

func (c *Container) initAuditRepository() error {
	if c.auditRepo != nil {
		return nil
	}
	if c.pgxPool == nil {
		return fmt.Errorf("database pool not initialized")
	}
	var err error
	c.auditRepo, err = persistence.NewAuditRepository(c.pgxPool)
	if err == nil {
		c.logger.Debug("initialized audit repository")
	}
	return err
}

func (c *Container) initClientStore() error {
	if c.clientStore != nil {
		return nil
	}
	var err error
	c.clientStore, err = infra_auth.NewFileClientStore(c.config.ClientCredentialsPath)
	if err == nil {
		c.logger.Debug("initialized client store", "path", c.config.ClientCredentialsPath)
	}
	return err
}

func (c *Container) initTokenStore() error {
	if c.tokenStore != nil {
		return nil
	}
	c.tokenStore = infra_auth.NewInMemoryTokenStore()
	c.logger.Debug("initialized in-memory token store")
	return nil
}

func (c *Container) initTokenManager() error {
	if c.tokenManager != nil {
		return nil
	}
	if c.tokenStore == nil {
		return fmt.Errorf("token store not initialized")
	}
	var err error
	c.tokenManager, err = infra_auth.NewTokenManager(c.config.BootstrapSecrets.JWTRSAPrivateKey, c.tokenStore, c.auditLogger)
	if err == nil {
		c.logger.Debug("initialized token manager")
	}
	return err
}

func (c *Container) Close() error {
	var errs []error
	if c.pgxPool != nil {
		c.pgxPool.Close()
		c.logger.Debug("closed database connection pool")
	}
	for name, provider := range c.kmsProviders {
		if closer, ok := provider.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				errs = append(errs, fmt.Errorf("failed to close KMS provider %s: %w", name, err))
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors during cleanup: %v", errs)
	}
	return nil
}

func (c *Container) GetS3KeyRepository(ctx context.Context) (domain.KeyRepository, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(c.config.AWS.Region))
	if err != nil {
		return nil, err
	}
	return persistence.NewS3Storage(awsCfg, c.config.AWS.S3Bucket, c.logger)
}


func ProvideDependencies(cfg *infra_config.Config) (map[string]kms.KMSProvider, domain.KeyRepository, domain.AuditRepository, domain.ClientStore, *infra_auth.TokenManager, error) {
	container := NewContainer(cfg, slog.Default())
	defer func() {
		if err := container.Close(); err != nil {
			slog.Error("failed to close container during legacy call", "error", err)
		}
	}()
	deps, err := container.GetDependencies(context.Background())
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	return deps.KMSProviders, deps.KeyRepo, deps.AuditRepo, deps.ClientStore, deps.TokenManager, nil
}