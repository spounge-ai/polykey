package cache

import (
	"context"
	"time"
)

// Reader defines the read-only operations for a cache.
// K is the key type, and V is the value type.
type Reader[K comparable, V any] interface {
	Get(ctx context.Context, key K) (V, bool)
}

// Writer defines the write-only operations for a cache.
type Writer[K comparable, V any] interface {
	Set(ctx context.Context, key K, value V, ttl time.Duration)
	Delete(ctx context.Context, key K)
}

// Store is the primary interface that defines the behavior of the cache.
// It combines read, write, and maintenance operations.
type Store[K comparable, V any] interface {
	Reader[K, V]
	Writer[K, V]
	Count() int
	Clear(ctx context.Context)
}
