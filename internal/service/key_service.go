package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/spounge-ai/polykey/internal/domain"
	app_errors "github.com/spounge-ai/polykey/internal/errors"
	"github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/infra/persistence"
	"github.com/spounge-ai/polykey/internal/kms"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

var (
	ErrInvalidRequest    = errors.New("invalid request")
	ErrInvalidKeyType    = errors.New("invalid key type")
	ErrKeyGenerationFail = errors.New("failed to generate cryptographic key")
	ErrEntropyValidationFail = errors.New("entropy validation failed")
	ErrMissingMetadata   = errors.New("key metadata is missing")
)

type KeyService interface {
	CreateKey(ctx context.Context, req *pk.CreateKeyRequest) (*pk.CreateKeyResponse, error)
	GetKey(ctx context.Context, req *pk.GetKeyRequest) (*pk.GetKeyResponse, error)
	ListKeys(ctx context.Context, req *pk.ListKeysRequest) (*pk.ListKeysResponse, error)
	RotateKey(ctx context.Context, req *pk.RotateKeyRequest) (*pk.RotateKeyResponse, error)
	RevokeKey(ctx context.Context, req *pk.RevokeKeyRequest) error
	UpdateKeyMetadata(ctx context.Context, req *pk.UpdateKeyMetadataRequest) error
	GetKeyMetadata(ctx context.Context, req *pk.GetKeyMetadataRequest) (*pk.GetKeyMetadataResponse, error)
}

type keyServiceImpl struct {
	keyRepo         domain.KeyRepository
	kmsProviders    map[string]kms.KMSProvider
	logger          *slog.Logger
	cfg             *config.Config
	errorClassifier *app_errors.ErrorClassifier
}

func NewKeyService(cfg *config.Config, keyRepo domain.KeyRepository, kmsProviders map[string]kms.KMSProvider, logger *slog.Logger, errorClassifier *app_errors.ErrorClassifier) KeyService {
	return &keyServiceImpl{
		cfg:             cfg,
		keyRepo:         keyRepo,
		kmsProviders:    kmsProviders,
		logger:          logger,
		errorClassifier: errorClassifier,
	}
}

func (s *keyServiceImpl) getKMSProvider(dataClassification string) (kms.KMSProvider, error) {
	// Determine provider based on the data classification (tier) of the key itself.
	providerName := "local"
	if dataClassification == string(domain.TierPro) || dataClassification == string(domain.TierEnterprise) {
		providerName = "aws"
	}

	provider, ok := s.kmsProviders[providerName]
	if !ok {
		return nil, fmt.Errorf("%s kms provider not found", providerName)
	}
	return provider, nil
}



func (s *keyServiceImpl) getKeyByRequest(ctx context.Context, keyID domain.KeyID, version int32) (*domain.Key, error) {
	var key *domain.Key
	var err error

	if version > 0 {
		key, err = s.keyRepo.GetKeyByVersion(ctx, keyID, version)
	} else {
		key, err = s.keyRepo.GetKey(ctx, keyID)
	}

	if err != nil {
		s.logger.ErrorContext(ctx, "[key_service.go:getKeyByRequest] Error from keyRepo", "error", err)
		if errors.Is(err, persistence.ErrKeyNotFound) {
			return nil, app_errors.ErrKeyNotFound
		}
		return nil, err
	}
	return key, nil
}
