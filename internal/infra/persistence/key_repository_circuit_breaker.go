package persistence

import (
	"context"
	"time"

	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/pkg/patterns/circuitbreaker"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// KeyRepositoryCircuitBreaker adds a circuit breaker to a KeyRepository.
// It uses multiple type-safe breakers to avoid runtime type assertions.
type KeyRepositoryCircuitBreaker struct {
	repo        domain.KeyRepository
	voidBreaker *circuitbreaker.Breaker[any] // Single breaker for all methods
}

// NewKeyRepositoryCircuitBreaker creates a new KeyRepository with a circuit breaker.
func NewKeyRepositoryCircuitBreaker(repo domain.KeyRepository, maxFailures int, resetTimeout time.Duration) domain.KeyRepository {
	opts := []circuitbreaker.Option[any]{
		circuitbreaker.WithResetTimeout[any](resetTimeout),
	}

	return &KeyRepositoryCircuitBreaker{
		repo:        repo,
		voidBreaker: circuitbreaker.New(maxFailures, opts...),
	}
}

func (cb *KeyRepositoryCircuitBreaker) GetKey(ctx context.Context, id domain.KeyID) (*domain.Key, error) {
	result, err := cb.voidBreaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return cb.repo.GetKey(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return result.(*domain.Key), nil
}

func (cb *KeyRepositoryCircuitBreaker) GetKeyByVersion(ctx context.Context, id domain.KeyID, version int32) (*domain.Key, error) {
	result, err := cb.voidBreaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return cb.repo.GetKeyByVersion(ctx, id, version)
	})
	if err != nil {
		return nil, err
	}
	return result.(*domain.Key), nil
}

func (cb *KeyRepositoryCircuitBreaker) GetKeyMetadata(ctx context.Context, id domain.KeyID) (*pk.KeyMetadata, error) {
	result, err := cb.voidBreaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return cb.repo.GetKeyMetadata(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return result.(*pk.KeyMetadata), nil
}

func (cb *KeyRepositoryCircuitBreaker) GetKeyMetadataByVersion(ctx context.Context, id domain.KeyID, version int32) (*pk.KeyMetadata, error) {
	result, err := cb.voidBreaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return cb.repo.GetKeyMetadataByVersion(ctx, id, version)
	})
	if err != nil {
		return nil, err
	}
	return result.(*pk.KeyMetadata), nil
}

func (cb *KeyRepositoryCircuitBreaker) CreateKey(ctx context.Context, key *domain.Key) error {
	_, err := cb.voidBreaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return nil, cb.repo.CreateKey(ctx, key)
	})
	return err
}

func (cb *KeyRepositoryCircuitBreaker) CreateBatchKeys(ctx context.Context, keys []*domain.Key) error {
	_, err := cb.voidBreaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return nil, cb.repo.CreateBatchKeys(ctx, keys)
	})
	return err
}

func (cb *KeyRepositoryCircuitBreaker) ListKeys(ctx context.Context, lastCreatedAt *time.Time, limit int) ([]*domain.Key, error) {
	result, err := cb.voidBreaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return cb.repo.ListKeys(ctx, lastCreatedAt, limit)
	})
	if err != nil {
		return nil, err
	}
	return result.([]*domain.Key), nil
}

func (cb *KeyRepositoryCircuitBreaker) UpdateKeyMetadata(ctx context.Context, id domain.KeyID, metadata *pk.KeyMetadata) error {
	_, err := cb.voidBreaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return nil, cb.repo.UpdateKeyMetadata(ctx, id, metadata)
	})
	return err
}

func (cb *KeyRepositoryCircuitBreaker) RotateKey(ctx context.Context, id domain.KeyID, newEncryptedDEK []byte) (*domain.Key, error) {
	result, err := cb.voidBreaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return cb.repo.RotateKey(ctx, id, newEncryptedDEK)
	})
	if err != nil {
		return nil, err
	}
	return result.(*domain.Key), nil
}

func (cb *KeyRepositoryCircuitBreaker) RevokeKey(ctx context.Context, id domain.KeyID) error {
	_, err := cb.voidBreaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return nil, cb.repo.RevokeKey(ctx, id)
	})
	return err
}

func (cb *KeyRepositoryCircuitBreaker) GetKeyVersions(ctx context.Context, id domain.KeyID) ([]*domain.Key, error) {
	result, err := cb.voidBreaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return cb.repo.GetKeyVersions(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return result.([]*domain.Key), nil
}

func (cb *KeyRepositoryCircuitBreaker) Exists(ctx context.Context, id domain.KeyID) (bool, error) {
	result, err := cb.voidBreaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return cb.repo.Exists(ctx, id)
	})
	if err != nil {
		return false, err
	}
	return result.(bool), nil
}

func (cb *KeyRepositoryCircuitBreaker) GetBatchKeys(ctx context.Context, ids []domain.KeyID) ([]*domain.Key, error) {
	result, err := cb.voidBreaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return cb.repo.GetBatchKeys(ctx, ids)
	})
	if err != nil {
		return nil, err
	}
	return result.([]*domain.Key), nil
}

func (cb *KeyRepositoryCircuitBreaker) GetBatchKeyMetadata(ctx context.Context, ids []domain.KeyID) ([]*pk.KeyMetadata, error) {
	result, err := cb.voidBreaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return cb.repo.GetBatchKeyMetadata(ctx, ids)
	})
	if err != nil {
		return nil, err
	}
	return result.([]*pk.KeyMetadata), nil
}

func (cb *KeyRepositoryCircuitBreaker) RevokeBatchKeys(ctx context.Context, ids []domain.KeyID) error {
	_, err := cb.voidBreaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return nil, cb.repo.RevokeBatchKeys(ctx, ids)
	})
	return err
}

func (cb *KeyRepositoryCircuitBreaker) UpdateBatchKeyMetadata(ctx context.Context, updates []*domain.Key) error {
	_, err := cb.voidBreaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return nil, cb.repo.UpdateBatchKeyMetadata(ctx, updates)
	})
	return err
}

