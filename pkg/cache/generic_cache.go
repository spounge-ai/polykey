package cache

import (
	"context"
	"sync"
	"time"
)

const (
	// DefaultTTL is the default time-to-live for cache items.
	DefaultTTL = 5 * time.Minute
	// DefaultCleanupInterval is the default interval for cleaning up expired items.
	DefaultCleanupInterval = 10 * time.Minute
)

// item represents a cached value, including its expiration time.
type item[V any] struct {
	value     V
	expiresAt time.Time
	permanent bool
}

// Cache is a thread-safe, generic cache with expiration and cleanup.
type Cache[K comparable, V any] struct {
	mu              sync.RWMutex
	items           map[K]item[V]
	defaultTTL      time.Duration
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
	onEvicted       func(K, V)
}

// Option is a functional option for configuring the cache.
type Option[K comparable, V any] func(*Cache[K, V])

// New creates a new cache with the given options.
func New[K comparable, V any](opts ...Option[K, V]) *Cache[K, V] {
	c := &Cache[K, V]{
		items:           make(map[K]item[V]),
		defaultTTL:      DefaultTTL,
		cleanupInterval: DefaultCleanupInterval,
		stopCleanup:     make(chan struct{}),
	}

	for _, opt := range opts {
		opt(c)
	}

	go c.cleanupLoop()

	return c
}

// WithDefaultTTL sets the default time-to-live for cache items.
func WithDefaultTTL[K comparable, V any](ttl time.Duration) Option[K, V] {
	return func(c *Cache[K, V]) {
		c.defaultTTL = ttl
	}
}

// WithCleanupInterval sets the interval for cleaning up expired items.
func WithCleanupInterval[K comparable, V any](interval time.Duration) Option[K, V] {
	return func(c *Cache[K, V]) {
		c.cleanupInterval = interval
	}
}

// WithEvictionCallback sets a function to be called when an item is evicted.
func WithEvictionCallback[K comparable, V any](onEvicted func(K, V)) Option[K, V] {
	return func(c *Cache[K, V]) {
		c.onEvicted = onEvicted
	}
}

// Set adds an item to the cache, overwriting any existing item.
// If ttl is 0, the default TTL is used. If ttl is -1, the item never expires.
func (c *Cache[K, V]) Set(ctx context.Context, key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiresAt time.Time
	permanent := false

	switch ttl {
	case -1: // No expiration
		permanent = true
	case 0: // Default TTL
		expiresAt = time.Now().Add(c.defaultTTL)
	default: // Custom TTL
		expiresAt = time.Now().Add(ttl)
	}

	c.items[key] = item[V]{
		value:     value,
		expiresAt: expiresAt,
		permanent: permanent,
	}
}

// Get retrieves an item from the cache.
// It returns the item's value and true if the item was found and has not expired.
// Otherwise, it returns the zero value for the value type and false.
func (c *Cache[K, V]) Get(ctx context.Context, key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cachedItem, found := c.items[key]
	if !found {
		var zeroV V
		return zeroV, false
	}

	if !cachedItem.permanent && time.Now().After(cachedItem.expiresAt) {
		// Item has expired, but we don't delete it here to avoid a lock upgrade.
		// The cleanup goroutine will handle deletion.
		var zeroV V
		return zeroV, false
	}

	return cachedItem.value, true
}

// Delete removes an item from the cache.
func (c *Cache[K, V]) Delete(ctx context.Context, key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.delete(key)
}

func (c *Cache[K, V]) delete(key K) {
	if item, found := c.items[key]; found {
		delete(c.items, key)
		if c.onEvicted != nil {
			c.onEvicted(key, item.value)
		}
	}
}

// Count returns the number of items in the cache.
func (c *Cache[K, V]) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Clear removes all items from the cache.
func (c *Cache[K, V]) Clear(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[K]item[V])
}

// Stop terminates the cleanup goroutine.
func (c *Cache[K, V]) Stop() {
	close(c.stopCleanup)
}

func (c *Cache[K, V]) cleanupLoop() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.deleteExpired()
		case <-c.stopCleanup:
			return
		}
	}
}

func (c *Cache[K, V]) deleteExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, cachedItem := range c.items {
		if !cachedItem.permanent && now.After(cachedItem.expiresAt) {
			c.delete(key)
		}
	}
}
