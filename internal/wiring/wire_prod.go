package wiring

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spounge-ai/polykey/internal/domain"
	app_errors "github.com/spounge-ai/polykey/internal/errors"
	infra_audit "github.com/spounge-ai/polykey/internal/infra/audit"
	infra_auth "github.com/spounge-ai/polykey/internal/infra/auth"
	infra_config "github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/infra/persistence"
	infra_secrets "github.com/spounge-ai/polykey/internal/infra/secrets"
	"github.com/spounge-ai/polykey/internal/kms"
	"github.com/spounge-ai/polykey/internal/service"
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
	authorizer   domain.Authorizer
	keyService   service.KeyService
	authService  service.AuthService
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
	AuditLogger  domain.AuditLogger
	ClientStore  domain.ClientStore
	TokenManager *infra_auth.TokenManager
	Authorizer   domain.Authorizer
	KeyService   service.KeyService
	AuthService  service.AuthService
}

func (c *Container) GetDependencies(ctx context.Context) (*Dependencies, error) {
	if err := c.initializeAll(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize dependencies: %w", err)
	}
	return &Dependencies{
		KMSProviders: c.kmsProviders,
		KeyRepo:      c.keyRepo,
		AuditRepo:    c.auditRepo,
		AuditLogger:  c.auditLogger,
		ClientStore:  c.clientStore,
		TokenManager: c.tokenManager,
		Authorizer:   c.authorizer,
		KeyService:   c.keyService,
		AuthService:  c.authService,
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
		func(context.Context) error { return c.initAuthorizer() },
		func(context.Context) error { return c.initKeyService() },
		func(context.Context) error { return c.initAuthService() },
	}
	for _, initFn := range initializers {
		if err := initFn(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (c *Container) initAuthorizer() error {
	if c.authorizer != nil {
		return nil
	}
	if c.keyRepo == nil {
		return fmt.Errorf("key repository not initialized")
	}
	if c.auditLogger == nil {
		return fmt.Errorf("audit logger not initialized")
	}
	c.authorizer = infra_auth.NewAuthorizer(c.config.Authorization, c.keyRepo, c.auditLogger)
	c.logger.Debug("initialized authorizer")
	return nil
}

func (c *Container) initAuditLogger() error {
	if c.auditLogger != nil {
		return nil
	}
	if c.auditRepo == nil {
		return fmt.Errorf("audit repository not initialized")
	}

	if c.config.Auditing.Asynchronous.Enabled {
		asyncConfig := infra_audit.AsyncAuditLoggerConfig{
			ChannelBufferSize: c.config.Auditing.Asynchronous.ChannelBufferSize,
			WorkerCount:       c.config.Auditing.Asynchronous.WorkerCount,
			BatchSize:         c.config.Auditing.Asynchronous.BatchSize,
			BatchTimeout:      c.config.Auditing.Asynchronous.BatchTimeout,
		}
		asyncLogger := infra_audit.NewAsyncAuditLogger(c.logger, c.auditRepo, asyncConfig)
		asyncLogger.Start()
		c.auditLogger = asyncLogger
		c.logger.Debug("initialized asynchronous audit logger")
	} else {
		c.auditLogger = infra_audit.NewAuditLogger(c.logger, c.auditRepo)
		c.logger.Debug("initialized synchronous audit logger")
	}

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
		dbConfig := infra_config.NeonDBConfig{URL: c.config.BootstrapSecrets.NeonDBURL}
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

	var secretProvider *infra_secrets.ParameterStore
	var awsCfg aws.Config
	var err error

	// Initialize AWS provider if configured
	if c.config.AWS.Enabled {
		awsCfg, err = config.LoadDefaultConfig(ctx, config.WithRegion(c.config.AWS.Region))
		if err != nil {
			return fmt.Errorf("failed to load AWS config: %w", err)
		}
		secretProvider = infra_secrets.NewParameterStore(awsCfg)

		// Fetch KMS Key ARN from the specified path
		kmsKeyARN, err := secretProvider.GetSecret(ctx, c.config.PolykeyAWSKMSKeyARNPath)
		if err != nil {
			return fmt.Errorf("failed to load AWS KMS Key ARN from %s: %w", c.config.PolykeyAWSKMSKeyARNPath, err)
		}

		c.kmsProviders["aws"] = kms.NewAWSKMSProvider(awsCfg, kmsKeyARN)
		c.logger.Debug("initialized AWS KMS provider", "region", c.config.AWS.Region)
	}

	// Fetch CA cert from the specified path and store it in TLS config
	if c.config.Server.TLS.Enabled && c.config.SpoungeCA != "" {
		if secretProvider == nil {
			// If AWS is not enabled, we still need to load a default config to get a secretProvider
			awsCfg, err = config.LoadDefaultConfig(ctx)
			if err != nil {
				return fmt.Errorf("failed to load default AWS config for CA cert: %w", err)
			}
			secretProvider = infra_secrets.NewParameterStore(awsCfg)
		}

		caCert, err := secretProvider.GetSecret(ctx, c.config.SpoungeCA)
		if err != nil {
			return fmt.Errorf("failed to load CA cert from %s: %w", c.config.SpoungeCA, err)
		}
		c.config.Server.TLS.ClientCAFile = caCert
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
	baseRepo, err := persistence.NewPSQLAdapter(c.pgxPool, c.logger)
	if err != nil {
		return err
	}

	// Wrap it with the cache decorator
	cachedRepo := persistence.NewCachedRepository(baseRepo, c.logger)

	// Check if the circuit breaker is enabled
	if c.config.Persistence.CircuitBreaker.Enabled {
		c.logger.Debug("wrapping key repository with circuit breaker")
		c.keyRepo = persistence.NewKeyRepositoryCircuitBreaker(
			cachedRepo,
			c.config.Persistence.CircuitBreaker.MaxFailures,
			c.config.Persistence.CircuitBreaker.ResetTimeout,
		)
	} else {
		c.keyRepo = cachedRepo
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

func (c *Container) initKeyService() error {
	if c.keyService != nil {
		return nil
	}
	if c.keyRepo == nil {
		return fmt.Errorf("key repository not initialized")
	}
	if c.kmsProviders == nil {
		return fmt.Errorf("kms providers not initialized")
	}
	if c.auditLogger == nil {
		return fmt.Errorf("audit logger not initialized")
	}
	errorClassifier := app_errors.NewErrorClassifier(c.logger)
	c.keyService = service.NewKeyService(c.config, c.keyRepo, c.kmsProviders, c.logger, errorClassifier, c.auditLogger)
	c.logger.Debug("initialized key service")
	return nil
}

func (c *Container) initAuthService() error {
	if c.authService != nil {
		return nil
	}
	if c.clientStore == nil {
		return fmt.Errorf("client store not initialized")
	}
	if c.tokenManager == nil {
		return fmt.Errorf("token manager not initialized")
	}
	c.authService = service.NewAuthService(c.clientStore, c.tokenManager, time.Hour)
	c.logger.Debug("initialized auth service")
	return nil
}

func (c *Container) Close() error {
	// Stop the audit logger first to ensure all events are flushed before dependencies close.
	if c.auditLogger != nil {
		if logger, ok := c.auditLogger.(interface{ Stop() }); ok {
			logger.Stop()
		}
	}

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


func ProvideDependencies(cfg *infra_config.Config) (map[string]kms.KMSProvider, domain.KeyRepository, domain.AuditRepository, domain.ClientStore, *infra_auth.TokenManager, domain.Authorizer, error) {
	container := NewContainer(cfg, slog.Default())
	defer func() {
		if err := container.Close(); err != nil {
			slog.Error("failed to close container during legacy call", "error", err)
		}
	}()
	deps, err := container.GetDependencies(context.Background())
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	return deps.KMSProviders, deps.KeyRepo, deps.AuditRepo, deps.ClientStore, deps.TokenManager, deps.Authorizer, nil
}