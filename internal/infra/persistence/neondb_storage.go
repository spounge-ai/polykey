package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/pkg/cache"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var ErrKeyNotFound = errors.New("key not found")

type NeonDBStorage struct {
	db               *pgxpool.Pool
	logger           *slog.Logger
	cache            cache.Store[string, *domain.Key]
	cacheIndex       map[string]map[string]struct{}
	cacheIndexMux    sync.RWMutex
	queryBuilderPool *sync.Pool
	stmtCache        map[string]*pgx.Conn
}

func NewNeonDBStorage(db *pgxpool.Pool, logger *slog.Logger) (*NeonDBStorage, error) {
	s := &NeonDBStorage{
		db:         db,
		logger:     logger,
		cacheIndex: make(map[string]map[string]struct{}),
		queryBuilderPool: &sync.Pool{
			New: func() interface{} {
				return &strings.Builder{}
			},
		},
		stmtCache: make(map[string]*pgx.Conn),
	}

	c := cache.New[string, *domain.Key](
		cache.WithDefaultTTL[string, *domain.Key](5*time.Minute),
		cache.WithCleanupInterval[string, *domain.Key](10*time.Minute),
		cache.WithEvictionCallback[string, *domain.Key](s.onCacheEvict),
	)

	s.cache = c

	return s, nil
}

func (s *NeonDBStorage) onCacheEvict(cacheKey string, key *domain.Key) {
	if key == nil {
		return
	}
	s.cacheIndexMux.Lock()
	defer s.cacheIndexMux.Unlock()

	keyIDStr := key.ID.String()
	if keys, ok := s.cacheIndex[keyIDStr]; ok {
		delete(keys, cacheKey)
		if len(keys) == 0 {
			delete(s.cacheIndex, keyIDStr)
		}
	}
}

func (s *NeonDBStorage) GetKey(ctx context.Context, id domain.KeyID) (*domain.Key, error) {
	cacheKey := s.getCacheKey(id, 0)
	if key, found := s.cache.Get(ctx, cacheKey); found {
		return key, nil
	}

	const query = `SELECT version, metadata, encrypted_dek, status, created_at, updated_at, revoked_at 
	               FROM keys WHERE id = $1 ORDER BY version DESC LIMIT 1`

	var key domain.Key
	var metadataRaw []byte

	err := s.db.QueryRow(ctx, query, id.String()).Scan(
		&key.Version, &metadataRaw, &key.EncryptedDEK,
		&key.Status, &key.CreatedAt, &key.UpdatedAt, &key.RevokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to query key %s: %w", id.String(), err)
	}

	if err := json.Unmarshal(metadataRaw, &key.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata for key %s: %w", id.String(), err)
	}

	key.ID = id
	s.storeInCache(ctx, cacheKey, &key)

	return &key, nil
}

func (s *NeonDBStorage) GetKeyByVersion(ctx context.Context, id domain.KeyID, version int32) (*domain.Key, error) {
	cacheKey := s.getCacheKey(id, version)
	if key, found := s.cache.Get(ctx, cacheKey); found {
		return key, nil
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
	s.storeInCache(ctx, cacheKey, &key)

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

	// No need to invalidate cache for a new key
	return nil
}

func (s *NeonDBStorage) ListKeys(ctx context.Context) ([]*domain.Key, error) {
	const query = `SELECT id, version, metadata, encrypted_dek, status, created_at, updated_at, revoked_at FROM keys`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query keys: %w", err)
	}
	defer rows.Close()

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

	s.invalidateCache(ctx, id)
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

	s.invalidateCache(ctx, id)

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
	const query = `UPDATE keys SET status = $1, revoked_at = $2 WHERE id = $3`

	result, err := s.db.Exec(ctx, query, domain.KeyStatusRevoked, time.Now(), id.String())
	if err != nil {
		return fmt.Errorf("failed to revoke key %s: %w", id.String(), err)
	}

	if result.RowsAffected() == 0 {
		return ErrKeyNotFound
	}

	s.invalidateCache(ctx, id)
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

func (s *NeonDBStorage) Exists(ctx context.Context, id domain.KeyID) (bool, error) {
	cacheKey := s.getCacheKey(id, 0)
	if _, ok := s.cache.Get(ctx, cacheKey); ok {
		return true, nil
	}

	const query = `SELECT EXISTS(SELECT 1 FROM keys WHERE id = $1 LIMIT 1)`

	var exists bool
	err := s.db.QueryRow(ctx, query, id.String()).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check key existence %s: %w", id.String(), err)
	}

	return exists, nil
}

func (s *NeonDBStorage) Close() error {
	if c, ok := s.cache.(interface{ Stop() }); ok {
		c.Stop()
	}
	return nil
}

func (s *NeonDBStorage) getCacheKey(id domain.KeyID, version int32) string {
	sb := s.queryBuilderPool.Get().(*strings.Builder)
	defer s.queryBuilderPool.Put(sb)
	sb.Reset()

	sb.WriteString(id.String())
	if version == 0 {
		sb.WriteString(":latest")
	} else {
		sb.WriteString(":v")
		sb.WriteString(strconv.Itoa(int(version)))
	}
	return sb.String()
}

func (s *NeonDBStorage) storeInCache(ctx context.Context, cacheKey string, k *domain.Key) {
	s.cache.Set(ctx, cacheKey, k, 0) // Use 0 for default TTL

	s.cacheIndexMux.Lock()
	defer s.cacheIndexMux.Unlock()
	keyIDStr := k.ID.String()
	if _, ok := s.cacheIndex[keyIDStr]; !ok {
		s.cacheIndex[keyIDStr] = make(map[string]struct{})
	}
	s.cacheIndex[keyIDStr][cacheKey] = struct{}{}
}

func (s *NeonDBStorage) invalidateCache(ctx context.Context, id domain.KeyID) {
	s.cacheIndexMux.RLock()
	keyIDStr := id.String()
	keysToDel, ok := s.cacheIndex[keyIDStr]
	s.cacheIndexMux.RUnlock()

	if !ok {
		return
	}

	for cacheKey := range keysToDel {
		s.cache.Delete(ctx, cacheKey)
	}
}
