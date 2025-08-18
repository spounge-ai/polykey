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
	repo            domain.KeyRepository
	getKeyBreaker   *circuitbreaker.Breaker[*domain.Key]
	listKeysBreaker *circuitbreaker.Breaker[[]*domain.Key]
	existsBreaker   *circuitbreaker.Breaker[bool]
	voidBreaker     *circuitbreaker.Breaker[any] // For methods returning only an error
}

// NewKeyRepositoryCircuitBreaker creates a new KeyRepository with a circuit breaker.
func NewKeyRepositoryCircuitBreaker(repo domain.KeyRepository, maxFailures int, resetTimeout time.Duration) domain.KeyRepository {
	// Shared options for all breakers
	opts := []circuitbreaker.Option[any]{
		circuitbreaker.WithResetTimeout[any](resetTimeout),
	}

	// It's safe to cast options for different generic types if they don't depend on the type T.
	// This is a bit of a workaround for Go's type system regarding generic functional options.
	getKeyOpts := []circuitbreaker.Option[*domain.Key]{
		circuitbreaker.WithResetTimeout[*domain.Key](resetTimeout),
	}
	listKeysOpts := []circuitbreaker.Option[[]*domain.Key]{
		circuitbreaker.WithResetTimeout[[]*domain.Key](resetTimeout),
	}
	existsOpts := []circuitbreaker.Option[bool]{
		circuitbreaker.WithResetTimeout[bool](resetTimeout),
	}

	return &KeyRepositoryCircuitBreaker{
		repo:            repo,
		getKeyBreaker:   circuitbreaker.New(maxFailures, getKeyOpts...),
		listKeysBreaker: circuitbreaker.New(maxFailures, listKeysOpts...),
		existsBreaker:   circuitbreaker.New(maxFailures, existsOpts...),
		voidBreaker:     circuitbreaker.New(maxFailures, opts...),
	}
}

func (cb *KeyRepositoryCircuitBreaker) GetKey(ctx context.Context, id domain.KeyID) (*domain.Key, error) {
	return cb.getKeyBreaker.Execute(ctx, func(ctx context.Context) (*domain.Key, error) {
		return cb.repo.GetKey(ctx, id)
	})
}

func (cb *KeyRepositoryCircuitBreaker) GetKeyByVersion(ctx context.Context, id domain.KeyID, version int32) (*domain.Key, error) {
	return cb.getKeyBreaker.Execute(ctx, func(ctx context.Context) (*domain.Key, error) {
		return cb.repo.GetKeyByVersion(ctx, id, version)
	})
}

func (cb *KeyRepositoryCircuitBreaker) CreateKey(ctx context.Context, key *domain.Key) (*domain.Key, error) {
	return cb.getKeyBreaker.Execute(ctx, func(ctx context.Context) (*domain.Key, error) {
		return cb.repo.CreateKey(ctx, key)
	})
}

func (cb *KeyRepositoryCircuitBreaker) CreateKeys(ctx context.Context, keys []*domain.Key) error {
	_, err := cb.voidBreaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return nil, cb.repo.CreateKeys(ctx, keys)
	})
	return err
}

func (cb *KeyRepositoryCircuitBreaker) ListKeys(ctx context.Context) ([]*domain.Key, error) {
	return cb.listKeysBreaker.Execute(ctx, func(ctx context.Context) ([]*domain.Key, error) {
		return cb.repo.ListKeys(ctx)
	})
}

func (cb *KeyRepositoryCircuitBreaker) UpdateKeyMetadata(ctx context.Context, id domain.KeyID, metadata *pk.KeyMetadata) error {
	_, err := cb.voidBreaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return nil, cb.repo.UpdateKeyMetadata(ctx, id, metadata)
	})
	return err
}

func (cb *KeyRepositoryCircuitBreaker) RotateKey(ctx context.Context, id domain.KeyID, newEncryptedDEK []byte) (*domain.Key, error) {
	return cb.getKeyBreaker.Execute(ctx, func(ctx context.Context) (*domain.Key, error) {
		return cb.repo.RotateKey(ctx, id, newEncryptedDEK)
	})
}

func (cb *KeyRepositoryCircuitBreaker) RevokeKey(ctx context.Context, id domain.KeyID) error {
	_, err := cb.voidBreaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return nil, cb.repo.RevokeKey(ctx, id)
	})
	return err
}

func (cb *KeyRepositoryCircuitBreaker) GetKeyVersions(ctx context.Context, id domain.KeyID) ([]*domain.Key, error) {
	return cb.listKeysBreaker.Execute(ctx, func(ctx context.Context) ([]*domain.Key, error) {
		return cb.repo.GetKeyVersions(ctx, id)
	})
}

func (cb *KeyRepositoryCircuitBreaker) Exists(ctx context.Context, id domain.KeyID) (bool, error) {
	return cb.existsBreaker.Execute(ctx, func(ctx context.Context) (bool, error) {
		return cb.repo.Exists(ctx, id)
	})
}
