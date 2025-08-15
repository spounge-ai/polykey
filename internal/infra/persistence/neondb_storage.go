package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spounge-ai/polykey/internal/domain"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var ErrKeyNotFound = errors.New("key not found")

const (
	defaultCacheTTL    = 5 * time.Minute
	maxCacheSize       = 10000 // Prevent unbounded cache growth
	cacheCleanupPeriod = 10 * time.Minute
)

type cacheEntry struct {
	key       *domain.Key
	expiresAt time.Time
}

type NeonDBStorage struct {
	db            *pgxpool.Pool
	logger        *slog.Logger
	cache         *sync.Map
	cacheIndex    map[string][]string
	cacheIndexMux sync.RWMutex
	cacheSize     int64

	// String builders for efficient query building
	queryBuilder strings.Builder
	queryMutex   sync.Mutex

	// Prepared statement cache
	stmtCache map[string]*pgx.Conn

	// Background cleanup
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

func NewNeonDBStorage(db *pgxpool.Pool, logger *slog.Logger) (*NeonDBStorage, error) {
	s := &NeonDBStorage{
		db:            db,
		logger:        logger,
		cache:         &sync.Map{},
		cacheIndex:    make(map[string][]string),
		stmtCache:     make(map[string]*pgx.Conn),
		cleanupTicker: time.NewTicker(cacheCleanupPeriod),
		stopCleanup:   make(chan struct{}),
	}

	// Start background cleanup goroutine
	go s.cleanupExpiredCache()

	return s, nil
}

// Background cleanup to prevent memory leaks
func (s *NeonDBStorage) cleanupExpiredCache() {
	for {
		select {
		case <-s.cleanupTicker.C:
			s.cleanupExpiredEntries()
		case <-s.stopCleanup:
			return
		}
	}
}

func (s *NeonDBStorage) cleanupExpiredEntries() {
	now := time.Now()
	var toDelete []string
	
	s.cache.Range(func(key, value interface{}) bool {
		entry := value.(cacheEntry)
		if now.After(entry.expiresAt) {
			toDelete = append(toDelete, key.(string))
		}
		return true
	})
	
	for _, key := range toDelete {
		s.cache.Delete(key)
	}
}

func (s *NeonDBStorage) GetKey(ctx context.Context, id domain.KeyID) (*domain.Key, error) {
	cacheKey := s.getCacheKey(id, 0)
	if entry, ok := s.loadFromCache(cacheKey); ok {
		return entry, nil
	}

	// Optimized for composite primary key (id, version) - uses index efficiently
	const query = `SELECT version, metadata, encrypted_dek, status, created_at, updated_at, revoked_at 
	               FROM keys WHERE id = $1 ORDER BY version DESC LIMIT 1`
	
	var key domain.Key
	var metadataRaw []byte

	err := s.db.QueryRow(ctx, query, id.String()).Scan(
		&key.Version, &metadataRaw, &key.EncryptedDEK, 
		&key.Status, &key.CreatedAt, &key.UpdatedAt, &key.RevokedAt)
	if err != nil {
		s.logger.ErrorContext(ctx, "[neondb_storage.go:GetKey] Error from QueryRow", "error", err)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to query key %s: %w", id.String(), err)
	}

	if err := json.Unmarshal(metadataRaw, &key.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata for key %s: %w", id.String(), err)
	}

	key.ID = id
	s.storeInCache(cacheKey, &key)

	return &key, nil
}

func (s *NeonDBStorage) GetKeyByVersion(ctx context.Context, id domain.KeyID, version int32) (*domain.Key, error) {
	cacheKey := s.getCacheKey(id, version)
	if entry, ok := s.loadFromCache(cacheKey); ok {
		return entry, nil
	}

	const query = `SELECT metadata, encrypted_dek, status, created_at, updated_at, revoked_at 
	               FROM keys WHERE id = $1 AND version = $2`
	
	var key domain.Key
	var metadataRaw []byte

	err := s.db.QueryRow(ctx, query, id.String(), version).Scan(
		&metadataRaw, &key.EncryptedDEK, &key.Status, 
		&key.CreatedAt, &key.UpdatedAt, &key.RevokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to query key %s version %d: %w", id.String(), version, err)
	}

	if err := json.Unmarshal(metadataRaw, &key.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata for key %s: %w", id.String(), err)
	}

	key.ID = id
	key.Version = version
	s.storeInCache(cacheKey, &key)

	return &key, nil
}

func (s *NeonDBStorage) CreateKey(ctx context.Context, key *domain.Key, isPremium bool) error {
	metadataRaw, err := json.Marshal(key.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	const query = `INSERT INTO keys (id, version, metadata, encrypted_dek, status, created_at, updated_at, is_premium) 
	               VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	
	_, err = s.db.Exec(ctx, query, 
		key.ID.String(), key.Version, metadataRaw, key.EncryptedDEK, 
		key.Status, key.CreatedAt, key.UpdatedAt, isPremium)
	if err != nil {
		return fmt.Errorf("failed to create key %s: %w", key.ID.String(), err)
	}

	s.invalidateCache(key.ID)
	return nil
}

func (s *NeonDBStorage) ListKeys(ctx context.Context) ([]*domain.Key, error) {
	const query = `SELECT id, version, metadata, encrypted_dek, status, created_at, updated_at, revoked_at FROM keys`
	
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query keys: %w", err)
	}
	defer rows.Close()

	// Pre-allocate slice with reasonable capacity
	keys := make([]*domain.Key, 0, 100)
	
	for rows.Next() {
		var key domain.Key
		var metadataRaw []byte
		var idStr string
		
		err := rows.Scan(&idStr, &key.Version, &metadataRaw, &key.EncryptedDEK, 
			&key.Status, &key.CreatedAt, &key.UpdatedAt, &key.RevokedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan key row: %w", err)
		}

		key.ID, err = domain.KeyIDFromString(idStr)
		if err != nil {
			return nil, fmt.Errorf("invalid key ID %s: %w", idStr, err)
		}

		if err := json.Unmarshal(metadataRaw, &key.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata for key %s: %w", idStr, err)
		}

		keys = append(keys, &key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return keys, nil
}

func (s *NeonDBStorage) UpdateKeyMetadata(ctx context.Context, id domain.KeyID, metadata *pk.KeyMetadata) error {
	metadataRaw, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// With composite PK (id, version), we need to update the latest version
	// This updates ALL versions - might want to be more specific
	const query = `UPDATE keys SET metadata = $1, updated_at = $2 WHERE id = $3 AND version = (
		SELECT MAX(version) FROM keys WHERE id = $3
	)`
	
	result, err := s.db.Exec(ctx, query, metadataRaw, time.Now(), id.String())
	if err != nil {
		return fmt.Errorf("failed to update key metadata %s: %w", id.String(), err)
	}
	
	if result.RowsAffected() == 0 {
		return ErrKeyNotFound
	}

	s.invalidateCache(id)
	return nil
}

func (s *NeonDBStorage) RotateKey(ctx context.Context, id domain.KeyID, newEncryptedDEK []byte) (*domain.Key, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if rErr := tx.Rollback(ctx); rErr != nil && 
			!errors.Is(rErr, context.Canceled) && 
			!errors.Is(rErr, pgx.ErrTxClosed) {
			// Only log unexpected rollback errors
			s.logger.Error("failed to rollback transaction", "error", rErr)
		}
	}()

	var currentVersion int32
	var metadataRaw []byte
	var isPremium bool
	
	const selectQuery = `SELECT version, metadata, is_premium FROM keys 
	                     WHERE id = $1 ORDER BY version DESC LIMIT 1`
	
	err = tx.QueryRow(ctx, selectQuery, id.String()).Scan(&currentVersion, &metadataRaw, &isPremium)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to get current key version: %w", err)
	}

	var metadata pk.KeyMetadata
	if err := json.Unmarshal(metadataRaw, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	newVersion := currentVersion + 1
	now := time.Now()
	metadata.Version = newVersion
	metadata.UpdatedAt = timestamppb.New(now)

	newMetadataRaw, err := json.Marshal(&metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated metadata: %w", err)
	}

	const insertQuery = `INSERT INTO keys (id, version, metadata, encrypted_dek, status, created_at, updated_at, is_premium) 
	                     VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	
	_, err = tx.Exec(ctx, insertQuery, 
		id.String(), newVersion, newMetadataRaw, newEncryptedDEK, 
		domain.KeyStatusActive, now, now, isPremium)
	if err != nil {
		return nil, fmt.Errorf("failed to insert new key version: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit rotation transaction: %w", err)
	}

	s.invalidateCache(id)

	return &domain.Key{
		ID:           id,
		Version:      newVersion,
		Metadata:     &metadata,
		EncryptedDEK: newEncryptedDEK,
		Status:       domain.KeyStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (s *NeonDBStorage) RevokeKey(ctx context.Context, id domain.KeyID) error {
	// With composite PK, this will revoke ALL versions of the key
	// You might want to revoke only the latest version instead
	const query = `UPDATE keys SET status = $1, revoked_at = $2 WHERE id = $3`
	
	result, err := s.db.Exec(ctx, query, domain.KeyStatusRevoked, time.Now(), id.String())
	if err != nil {
		return fmt.Errorf("failed to revoke key %s: %w", id.String(), err)
	}
	
	if result.RowsAffected() == 0 {
		return ErrKeyNotFound
	}

	s.invalidateCache(id)
	return nil
}

func (s *NeonDBStorage) GetKeyVersions(ctx context.Context, id domain.KeyID) ([]*domain.Key, error) {
	const query = `SELECT id, version, metadata, encrypted_dek, status, created_at, updated_at, revoked_at 
	               FROM keys WHERE id = $1 ORDER BY version DESC`
	
	rows, err := s.db.Query(ctx, query, id.String())
	if err != nil {
		return nil, fmt.Errorf("failed to query key versions: %w", err)
	}
	defer rows.Close()

	// Pre-allocate with reasonable capacity
	keys := make([]*domain.Key, 0, 10)
	
	for rows.Next() {
		var key domain.Key
		var metadataRaw []byte
		var idStr string
		
		err := rows.Scan(&idStr, &key.Version, &metadataRaw, &key.EncryptedDEK, 
			&key.Status, &key.CreatedAt, &key.UpdatedAt, &key.RevokedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan key version row: %w", err)
		}

		key.ID, err = domain.KeyIDFromString(idStr)
		if err != nil {
			return nil, fmt.Errorf("invalid key ID %s: %w", idStr, err)
		}

		if err := json.Unmarshal(metadataRaw, &key.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		keys = append(keys, &key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over key versions: %w", err)
	}

	return keys, nil
}

// FIX: The main issue - Exists should not call GetKey
func (s *NeonDBStorage) Exists(ctx context.Context, id domain.KeyID) (bool, error) {
	// Check cache first
	cacheKey := s.getCacheKey(id, 0)
	if _, ok := s.loadFromCache(cacheKey); ok {
		return true, nil
	}
	
	// Simple existence check - don't fetch full key data
	const query = `SELECT EXISTS(SELECT 1 FROM keys WHERE id = $1 LIMIT 1)`
	
	var exists bool
	err := s.db.QueryRow(ctx, query, id.String()).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check key existence %s: %w", id.String(), err)
	}
	
	return exists, nil
}

// Graceful shutdown
func (s *NeonDBStorage) Close() error {
	if s.stopCleanup != nil {
		close(s.stopCleanup)
	}
	if s.cleanupTicker != nil {
		s.cleanupTicker.Stop()
	}
	return nil
}

// Optimized cache methods
func (s *NeonDBStorage) getCacheKey(id domain.KeyID, version int32) string {
	// Use string builder for efficient concatenation
	s.queryMutex.Lock()
	defer s.queryMutex.Unlock()
	
	s.queryBuilder.Reset()
	s.queryBuilder.WriteString(id.String())
	
	if version == 0 {
		s.queryBuilder.WriteString(":latest")
	} else {
		s.queryBuilder.WriteString(":v")
		s.queryBuilder.WriteString(fmt.Sprintf("%d", version))
	}
	
	return s.queryBuilder.String()
}

func (s *NeonDBStorage) loadFromCache(key string) (*domain.Key, bool) {
	entry, ok := s.cache.Load(key)
	if !ok {
		return nil, false
	}

	cached := entry.(cacheEntry)
	if time.Now().After(cached.expiresAt) {
		s.cache.Delete(key)
		s.decrementCacheSize()
		return nil, false
	}

	return cached.key, true
}

func (s *NeonDBStorage) storeInCache(cacheKey string, k *domain.Key) {
	// Prevent unbounded cache growth
	if s.cacheSize >= maxCacheSize {
		return
	}
	
	entry := cacheEntry{
		key:       k,
		expiresAt: time.Now().Add(defaultCacheTTL),
	}
	
	// Only increment if this is a new key
	if _, loaded := s.cache.LoadOrStore(cacheKey, entry); !loaded {
		s.incrementCacheSize()
		
		s.cacheIndexMux.Lock()
		keyIDStr := k.ID.String()
		s.cacheIndex[keyIDStr] = append(s.cacheIndex[keyIDStr], cacheKey)
		s.cacheIndexMux.Unlock()
	}
}

func (s *NeonDBStorage) incrementCacheSize() {
	// Atomic increment would be better, but this is simpler for now
	s.cacheSize++
}

func (s *NeonDBStorage) decrementCacheSize() {
	if s.cacheSize > 0 {
		s.cacheSize--
	}
}

func (s *NeonDBStorage) invalidateCache(id domain.KeyID) {
	s.cacheIndexMux.Lock()
	defer s.cacheIndexMux.Unlock()

	keyIDStr := id.String()
	cacheKeys, ok := s.cacheIndex[keyIDStr]
	if !ok {
		return
	}

	for _, cacheKey := range cacheKeys {
		if _, existed := s.cache.LoadAndDelete(cacheKey); existed {
			s.decrementCacheSize()
		}
	}

	delete(s.cacheIndex, keyIDStr)
}