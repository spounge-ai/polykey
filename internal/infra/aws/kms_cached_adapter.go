package aws

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/spounge-ai/polykey/internal/domain"
)

// KMSCachedAdapter adds a caching layer around the real KMS adapter.
type KMSCachedAdapter struct {
	next     domain.KMSService
	dekCache *cache.Cache
}

// NewKMSCachedAdapter creates a new KMSCachedAdapter.
func NewKMSCachedAdapter(next domain.KMSService) *KMSCachedAdapter {
	return &KMSCachedAdapter{
		next:     next,
		dekCache: cache.New(5*time.Minute, 10*time.Minute),
	}
}

// EncryptDEK encrypts a Data Encryption Key (DEK) using the specified master key in AWS KMS.
func (a *KMSCachedAdapter) EncryptDEK(ctx context.Context, plaintextDEK []byte, masterKeyID string) ([]byte, error) {
	return a.next.EncryptDEK(ctx, plaintextDEK, masterKeyID)
}

// DecryptDEK decrypts a Data Encryption Key (DEK) using AWS KMS.
func (a *KMSCachedAdapter) DecryptDEK(ctx context.Context, encryptedDEK []byte, masterKeyID string) ([]byte, error) {
	cacheKey := base64.StdEncoding.EncodeToString(encryptedDEK)

	if plaintextDEK, found := a.dekCache.Get(cacheKey); found {
		return plaintextDEK.([]byte), nil
	}

	plaintextDEK, err := a.next.DecryptDEK(ctx, encryptedDEK, masterKeyID)
	if err != nil {
		return nil, err
	}

	a.dekCache.Set(cacheKey, plaintextDEK, cache.DefaultExpiration)

	return plaintextDEK, nil
}
