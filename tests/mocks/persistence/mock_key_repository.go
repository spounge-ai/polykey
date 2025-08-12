package persistence

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/spounge-ai/polykey/internal/domain"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// InMemoryKeyRepository is an in-memory implementation of the KeyRepository interface for testing.
type InMemoryKeyRepository struct {
	keys *sync.Map
}

// NewInMemoryKeyRepository creates a new InMemoryKeyRepository.
func NewInMemoryKeyRepository() *InMemoryKeyRepository {
	return &InMemoryKeyRepository{
		keys: &sync.Map{},
	}
}

func (r *InMemoryKeyRepository) GetKey(ctx context.Context, id domain.KeyID) (*domain.Key, error) {
	val, ok := r.keys.Load(id.String())
	if !ok {
		return nil, fmt.Errorf("key not found")
	}
	return val.(*domain.Key), nil
}

func (r *InMemoryKeyRepository) GetKeyByVersion(ctx context.Context, id domain.KeyID, version int32) (*domain.Key, error) {
	// This is a simplified implementation. A real implementation would need to store versions.
	return r.GetKey(ctx, id)
}

func (r *InMemoryKeyRepository) CreateKey(ctx context.Context, key *domain.Key, isPremium bool) error {
	r.keys.Store(key.ID.String(), key)
	return nil
}

func (r *InMemoryKeyRepository) ListKeys(ctx context.Context) ([]*domain.Key, error) {
	var keys []*domain.Key
	r.keys.Range(func(key, value interface{}) bool {
		keys = append(keys, value.(*domain.Key))
		return true
	})
	return keys, nil
}

func (r *InMemoryKeyRepository) UpdateKeyMetadata(ctx context.Context, id domain.KeyID, metadata *pk.KeyMetadata) error {
	key, err := r.GetKey(ctx, id)
	if err != nil {
		return err
	}
	key.Metadata = metadata
	r.keys.Store(id.String(), key)
	return nil
}

func (r *InMemoryKeyRepository) RotateKey(ctx context.Context, id domain.KeyID, newEncryptedDEK []byte) (*domain.Key, error) {
	key, err := r.GetKey(ctx, id)
	if err != nil {
		return nil, err
	}
	key.Version++
	key.EncryptedDEK = newEncryptedDEK
	key.UpdatedAt = time.Now()
	r.keys.Store(id.String(), key)
	return key, nil
}

func (r *InMemoryKeyRepository) RevokeKey(ctx context.Context, id domain.KeyID) error {
	key, err := r.GetKey(ctx, id)
	if err != nil {
		return err
	}
	key.Status = domain.KeyStatusRevoked
	key.UpdatedAt = time.Now()
	r.keys.Store(id.String(), key)
	return nil
}

func (r *InMemoryKeyRepository) GetKeyVersions(ctx context.Context, id domain.KeyID) ([]*domain.Key, error) {
	// This is a simplified implementation. A real implementation would need to store versions.
	key, err := r.GetKey(ctx, id)
	if err != nil {
		return nil, err
	}
	return []*domain.Key{key}, nil
}

func (r *InMemoryKeyRepository) Exists(ctx context.Context, id domain.KeyID) (bool, error) {
	_, ok := r.keys.Load(id.String())
	return ok, nil
}
