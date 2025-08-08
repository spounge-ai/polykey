package aws

import (
	"context"
	"encoding/base64"
	"sync"
	"time"

	"github.com/spounge-ai/polykey/internal/domain"
)

type cacheItem struct {
	value      []byte
	expiration int64
}

type KMSCachedAdapter struct {
	next     domain.KMSService
	dekCache map[string]cacheItem
	mu       sync.RWMutex
	ttl      time.Duration
}

func NewKMSCachedAdapter(next domain.KMSService, ttl time.Duration) *KMSCachedAdapter {
	adapter := &KMSCachedAdapter{
		next:     next,
		dekCache: make(map[string]cacheItem),
		ttl:      ttl,
	}
	go adapter.cleanupLoop(ttl * 2) // Cleanup interval is twice the TTL
	return adapter
}

// cleanupLoop periodically removes expired items from the cache.
func (a *KMSCachedAdapter) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		a.mu.Lock()
		for key, item := range a.dekCache {
			if time.Now().UnixNano() > item.expiration {
				delete(a.dekCache, key)
			}
		}
		a.mu.Unlock()
	}
}

// EncryptDEK encrypts a Data Encryption Key (DEK) using the specified master key in AWS KMS.
func (a *KMSCachedAdapter) EncryptDEK(ctx context.Context, plaintextDEK []byte, masterKeyID string) ([]byte, error) {
	return a.next.EncryptDEK(ctx, plaintextDEK, masterKeyID)
}

// DecryptDEK decrypts a Data Encryption Key (DEK) using AWS KMS, with caching.
func (a *KMSCachedAdapter) DecryptDEK(ctx context.Context, encryptedDEK []byte, masterKeyID string) ([]byte, error) {
	cacheKey := base64.StdEncoding.EncodeToString(encryptedDEK)

	a.mu.RLock()
	item, found := a.dekCache[cacheKey]
	a.mu.RUnlock()

	if found && time.Now().UnixNano() < item.expiration {
		return item.value, nil
	}

	plaintextDEK, err := a.next.DecryptDEK(ctx, encryptedDEK, masterKeyID)
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	a.dekCache[cacheKey] = cacheItem{
		value:      plaintextDEK,
		expiration: time.Now().Add(a.ttl).UnixNano(),
	}
	a.mu.Unlock()

	return plaintextDEK, nil
}
