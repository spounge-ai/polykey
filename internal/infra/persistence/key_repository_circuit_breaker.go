package persistence

import (
	"context"

	"github.com/spounge-ai/polykey/internal/domain"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// KeyRepositoryCircuitBreaker adds a circuit breaker to a KeyRepository.
type KeyRepositoryCircuitBreaker struct {
	repo           domain.KeyRepository
	circuitBreaker *CircuitBreaker
}

// NewKeyRepositoryCircuitBreaker creates a new KeyRepository with a circuit breaker.
func NewKeyRepositoryCircuitBreaker(repo domain.KeyRepository, cb *CircuitBreaker) domain.KeyRepository {
	return &KeyRepositoryCircuitBreaker{
		repo:           repo,
		circuitBreaker: cb,
	}
}

func (cb *KeyRepositoryCircuitBreaker) GetKey(ctx context.Context, id domain.KeyID) (key *domain.Key, err error) {
	err = cb.circuitBreaker.Execute(ctx, func() error {
		key, err = cb.repo.GetKey(ctx, id)
		return err
	})
	return
}

func (cb *KeyRepositoryCircuitBreaker) GetKeyByVersion(ctx context.Context, id domain.KeyID, version int32) (key *domain.Key, err error) {
	err = cb.circuitBreaker.Execute(ctx, func() error {
		key, err = cb.repo.GetKeyByVersion(ctx, id, version)
		return err
	})
	return
}

func (cb *KeyRepositoryCircuitBreaker) CreateKey(ctx context.Context, key *domain.Key, isPremium bool) (err error) {
	return cb.circuitBreaker.Execute(ctx, func() error {
		return cb.repo.CreateKey(ctx, key, isPremium)
	})
}

func (cb *KeyRepositoryCircuitBreaker) ListKeys(ctx context.Context) (keys []*domain.Key, err error) {
	err = cb.circuitBreaker.Execute(ctx, func() error {
		keys, err = cb.repo.ListKeys(ctx)
		return err
	})
	return
}

func (cb *KeyRepositoryCircuitBreaker) UpdateKeyMetadata(ctx context.Context, id domain.KeyID, metadata *pk.KeyMetadata) (err error) {
	return cb.circuitBreaker.Execute(ctx, func() error {
		return cb.repo.UpdateKeyMetadata(ctx, id, metadata)
	})
}

func (cb *KeyRepositoryCircuitBreaker) RotateKey(ctx context.Context, id domain.KeyID, newEncryptedDEK []byte) (key *domain.Key, err error) {
	err = cb.circuitBreaker.Execute(ctx, func() error {
		key, err = cb.repo.RotateKey(ctx, id, newEncryptedDEK)
		return err
	})
	return
}

func (cb *KeyRepositoryCircuitBreaker) RevokeKey(ctx context.Context, id domain.KeyID) (err error) {
	return cb.circuitBreaker.Execute(ctx, func() error {
		return cb.repo.RevokeKey(ctx, id)
	})
}

func (cb *KeyRepositoryCircuitBreaker) GetKeyVersions(ctx context.Context, id domain.KeyID) (keys []*domain.Key, err error) {
	err = cb.circuitBreaker.Execute(ctx, func() error {
		keys, err = cb.repo.GetKeyVersions(ctx, id)
		return err
	})
	return
}

func (cb *KeyRepositoryCircuitBreaker) Exists(ctx context.Context, id domain.KeyID) (exists bool, err error) {
	err = cb.circuitBreaker.Execute(ctx, func() error {
		exists, err = cb.repo.Exists(ctx, id)
		return err
	})
	return
}
