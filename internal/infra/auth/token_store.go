package auth

import (
	"context"
	"time"

	"github.com/spounge-ai/polykey/pkg/cache"
)

// TokenStore defines the interface for storing revoked tokens.
// This abstraction allows for different implementations (e.g., in-memory, Redis).
type TokenStore interface {
	// Revoke adds a token to the revocation list.
	// The TTL should be the remaining validity of the token.
	Revoke(ctx context.Context, tokenID string, ttl time.Duration)
	// IsRevoked checks if a token has been revoked.
	IsRevoked(ctx context.Context, tokenID string) bool
}

// NewInMemoryTokenStore creates a new in-memory token store.
func NewInMemoryTokenStore() TokenStore {
	return &inMemoryTokenStore{
		store: cache.New(
			cache.WithCleanupInterval[string, struct{}](10 * time.Minute),
		),
	}
}

type inMemoryTokenStore struct {
	store cache.Store[string, struct{}]
}

func (s *inMemoryTokenStore) Revoke(ctx context.Context, tokenID string, ttl time.Duration) {
	s.store.Set(ctx, tokenID, struct{}{}, ttl)
}

func (s *inMemoryTokenStore) IsRevoked(ctx context.Context, tokenID string) bool {
	_, found := s.store.Get(ctx, tokenID)
	return found
}
