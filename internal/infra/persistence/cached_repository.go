package persistence

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/pkg/cache"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

const (
	// Cache configuration
	defaultCacheTTL     = 5 * time.Minute
	cacheCleanupInterval = 10 * time.Minute
)

// CachedRepository is a decorator for a KeyRepository that adds a caching layer.
type CachedRepository struct {
	repo          domain.KeyRepository
	cache         cache.Store[string, *domain.Key]
	cacheIndex    map[string]map[string]struct{}
	cacheIndexMux sync.RWMutex
	optimizer     *QueryOptimizer
	logger        *slog.Logger
}

// NewCachedRepository creates a new CachedRepository.
func NewCachedRepository(repo domain.KeyRepository, logger *slog.Logger) *CachedRepository {
	cr := &CachedRepository{
		repo:       repo,
		cacheIndex: make(map[string]map[string]struct{}),
		optimizer:  NewQueryOptimizer(),
		logger:     logger,
	}

	c := cache.New[string, *domain.Key](
		cache.WithDefaultTTL[string, *domain.Key](defaultCacheTTL),
		cache.WithCleanupInterval[string, *domain.Key](cacheCleanupInterval),
		cache.WithEvictionCallback[string, *domain.Key](cr.onCacheEvict),
	)
	cr.cache = c

	return cr
}

func (cr *CachedRepository) onCacheEvict(cacheKey string, key *domain.Key) {
	if key == nil {
		return
	}

	cr.cacheIndexMux.Lock()
	keyIDStr := key.ID.String()
	if keys, ok := cr.cacheIndex[keyIDStr]; ok {
		delete(keys, cacheKey)
		if len(keys) == 0 {
			delete(cr.cacheIndex, keyIDStr)
		}
	}
	cr.cacheIndexMux.Unlock()
}

func (cr *CachedRepository) GetKey(ctx context.Context, id domain.KeyID) (*domain.Key, error) {
	cacheKey := cr.getCacheKey(id, 0)
	if key, found := cr.cache.Get(ctx, cacheKey); found {
		return key, nil
	}

	key, err := cr.repo.GetKey(ctx, id)
	if err != nil {
		return nil, err
	}

	cr.storeInCache(cacheKey, key)
	return key, nil
}

func (cr *CachedRepository) GetKeyByVersion(ctx context.Context, id domain.KeyID, version int32) (*domain.Key, error) {
	cacheKey := cr.getCacheKey(id, version)
	if key, found := cr.cache.Get(ctx, cacheKey); found {
		return key, nil
	}

	key, err := cr.repo.GetKeyByVersion(ctx, id, version)
	if err != nil {
		return nil, err
	}

	cr.storeInCache(cacheKey, key)
	return key, nil
}

func (cr *CachedRepository) CreateKey(ctx context.Context, key *domain.Key) error {
	err := cr.repo.CreateKey(ctx, key)
	if err == nil {
		cr.invalidateCache(key.ID)
	}
	return err
}

func (cr *CachedRepository) ListKeys(ctx context.Context) ([]*domain.Key, error) {
	// Caching for ListKeys is complex and often not beneficial without proper invalidation strategies.
	// For now, we bypass the cache for this operation.
	return cr.repo.ListKeys(ctx)
}

func (cr *CachedRepository) UpdateKeyMetadata(ctx context.Context, id domain.KeyID, metadata *pk.KeyMetadata) error {
	err := cr.repo.UpdateKeyMetadata(ctx, id, metadata)
	if err == nil {
		cr.invalidateCache(id)
	}
	return err
}

func (cr *CachedRepository) RotateKey(ctx context.Context, id domain.KeyID, newEncryptedDEK []byte) (*domain.Key, error) {
	rotatedKey, err := cr.repo.RotateKey(ctx, id, newEncryptedDEK)
	if err == nil {
		cr.invalidateCache(id)
	}
	return rotatedKey, err
}

func (cr *CachedRepository) RevokeKey(ctx context.Context, id domain.KeyID) error {
	err := cr.repo.RevokeKey(ctx, id)
	if err == nil {
		cr.invalidateCache(id)
	}
	return err
}

func (cr *CachedRepository) GetKeyVersions(ctx context.Context, id domain.KeyID) ([]*domain.Key, error) {
	// Bypassing cache for simplicity.
	return cr.repo.GetKeyVersions(ctx, id)
}

func (cr *CachedRepository) Exists(ctx context.Context, id domain.KeyID) (bool, error) {
	cacheKey := cr.getCacheKey(id, 0)
	if _, found := cr.cache.Get(ctx, cacheKey); found {
		return true, nil
	}
	return cr.repo.Exists(ctx, id)
}

// Helper methods

func (cr *CachedRepository) getCacheKey(id domain.KeyID, version int32) string {
	sb := cr.optimizer.GetBuilder()
	defer cr.optimizer.PutBuilder(sb)

	sb.WriteString(id.String())
	if version == 0 {
		sb.WriteString(":latest")
	} else {
		sb.WriteString(":v")
		sb.WriteString(strconv.Itoa(int(version)))
	}
	return sb.String()
}

func (cr *CachedRepository) storeInCache(cacheKey string, k *domain.Key) {
	cr.cache.Set(context.Background(), cacheKey, k, 0)

	cr.cacheIndexMux.Lock()
	keyIDStr := k.ID.String()
	if _, ok := cr.cacheIndex[keyIDStr]; !ok {
		cr.cacheIndex[keyIDStr] = make(map[string]struct{})
	}
	cr.cacheIndex[keyIDStr][cacheKey] = struct{}{}
	cr.cacheIndexMux.Unlock()
}

func (cr *CachedRepository) invalidateCache(id domain.KeyID) {
	cr.cacheIndexMux.RLock()
	keyIDStr := id.String()
	keysToDel := make(map[string]struct{})
	if keys, ok := cr.cacheIndex[keyIDStr]; ok {
		for k := range keys {
			keysToDel[k] = struct{}{}
		}
	}
	cr.cacheIndexMux.RUnlock()

	for cacheKey := range keysToDel {
		cr.cache.Delete(context.Background(), cacheKey)
	}
}