package persistence

import (
	"context"
	"time"

	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/pkg/patterns/circuitbreaker"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// KeyRepositoryCircuitBreaker adds a circuit breaker to a KeyRepository.
type KeyRepositoryCircuitBreaker struct {
	repo    domain.KeyRepository
	breaker *circuitbreaker.Breaker[any]
}

// NewKeyRepositoryCircuitBreaker creates a new KeyRepository with a circuit breaker.
func NewKeyRepositoryCircuitBreaker(repo domain.KeyRepository, maxFailures int, resetTimeout time.Duration) domain.KeyRepository {
	return &KeyRepositoryCircuitBreaker{
		repo:    repo,
		breaker: circuitbreaker.New[any](maxFailures, resetTimeout),
	}
}

func (cb *KeyRepositoryCircuitBreaker) GetKey(ctx context.Context, id domain.KeyID) (*domain.Key, error) {
	res, err := cb.breaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return cb.repo.GetKey(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return res.(*domain.Key), nil
}

func (cb *KeyRepositoryCircuitBreaker) GetKeyByVersion(ctx context.Context, id domain.KeyID, version int32) (*domain.Key, error) {
	res, err := cb.breaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return cb.repo.GetKeyByVersion(ctx, id, version)
	})
	if err != nil {
		return nil, err
	}
	return res.(*domain.Key), nil
}

func (cb *KeyRepositoryCircuitBreaker) CreateKey(ctx context.Context, key *domain.Key, isPremium bool) error {
	_, err := cb.breaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return nil, cb.repo.CreateKey(ctx, key, isPremium)
	})
	return err
}

func (cb *KeyRepositoryCircuitBreaker) ListKeys(ctx context.Context) ([]*domain.Key, error) {
	res, err := cb.breaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return cb.repo.ListKeys(ctx)
	})
	if err != nil {
		return nil, err
	}
	return res.([]*domain.Key), nil
}

func (cb *KeyRepositoryCircuitBreaker) UpdateKeyMetadata(ctx context.Context, id domain.KeyID, metadata *pk.KeyMetadata) error {
	_, err := cb.breaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return nil, cb.repo.UpdateKeyMetadata(ctx, id, metadata)
	})
	return err
}

func (cb *KeyRepositoryCircuitBreaker) RotateKey(ctx context.Context, id domain.KeyID, newEncryptedDEK []byte) (*domain.Key, error) {
	res, err := cb.breaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return cb.repo.RotateKey(ctx, id, newEncryptedDEK)
	})
	if err != nil {
		return nil, err
	}
	return res.(*domain.Key), nil
}

func (cb *KeyRepositoryCircuitBreaker) RevokeKey(ctx context.Context, id domain.KeyID) error {
	_, err := cb.breaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return nil, cb.repo.RevokeKey(ctx, id)
	})
	return err
}

func (cb *KeyRepositoryCircuitBreaker) GetKeyVersions(ctx context.Context, id domain.KeyID) ([]*domain.Key, error) {
	res, err := cb.breaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return cb.repo.GetKeyVersions(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return res.([]*domain.Key), nil
}

func (cb *KeyRepositoryCircuitBreaker) Exists(ctx context.Context, id domain.KeyID) (bool, error) {
	res, err := cb.breaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return cb.repo.Exists(ctx, id)
	})
	if err != nil {
		return false, err
	}
	return res.(bool), nil
}