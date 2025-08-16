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
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spounge-ai/polykey/internal/constants"
	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/pkg/cache"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	ErrKeyNotFound     = errors.New("key not found")
	ErrInvalidVersion  = errors.New("invalid version")
	ErrKeyAlreadyExists = errors.New("key already exists")
)

// Prepared statement names
const (
	stmtGetLatestKey    = "get_latest_key"
	stmtGetKeyByVersion = "get_key_by_version"
	stmtCreateKey       = "create_key"
	stmtUpdateMetadata  = "update_metadata"
	stmtRevokeKey       = "revoke_key"
	stmtCheckExists     = "check_exists"
	stmtGetVersions     = "get_versions"
	stmtListKeys        = "list_keys"
)

type NeonDBStorage struct {
	db               *pgxpool.Pool
	logger           *slog.Logger
	cache            cache.Store[string, *domain.Key]
	cacheIndex       map[string]map[string]struct{}
	cacheIndexMux    sync.RWMutex
	queryBuilderPool *sync.Pool
	prepared         bool
	prepareMux       sync.Once
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
	}

	c := cache.New[string, *domain.Key](
		cache.WithDefaultTTL[string, *domain.Key](5*time.Minute),
		cache.WithCleanupInterval[string, *domain.Key](10*time.Minute),
		cache.WithEvictionCallback[string, *domain.Key](s.onCacheEvict),
	)

	s.cache = c

	return s, nil
}

// prepareStatements prepares commonly used SQL statements for better performance
func (s *NeonDBStorage) prepareStatements(ctx context.Context) error {
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection for statement preparation: %w", err)
	}
	defer conn.Release()

	statements := map[string]string{
		stmtGetLatestKey: `
			SELECT version, metadata, encrypted_dek, status, storage_type, created_at, updated_at, revoked_at 
			FROM keys 
			WHERE id = $1::uuid 
			ORDER BY version DESC 
			LIMIT 1`,

		stmtGetKeyByVersion: `
			SELECT metadata, encrypted_dek, status, storage_type, created_at, updated_at, revoked_at 
			FROM keys 
			WHERE id = $1::uuid AND version = $2`,

		stmtCreateKey: `
			INSERT INTO keys (id, version, metadata, encrypted_dek, status, storage_type, created_at, updated_at) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,

		stmtUpdateMetadata: `
			UPDATE keys 
			SET metadata = $1, updated_at = $2 
			WHERE id = $3::uuid AND version = (
				SELECT MAX(version) FROM keys WHERE id = $3::uuid
			)`,

		stmtRevokeKey: `
			UPDATE keys 
			SET status = $1, revoked_at = $2 
			WHERE id = $3::uuid`,

		stmtCheckExists: `
			SELECT EXISTS(SELECT 1 FROM keys WHERE id = $1::uuid LIMIT 1)`,

		stmtGetVersions: `
			SELECT version, metadata, encrypted_dek, status, storage_type, created_at, updated_at, revoked_at 
			FROM keys 
			WHERE id = $1::uuid 
			ORDER BY version DESC`,

		stmtListKeys: `
			WITH latest_keys AS (
				SELECT DISTINCT ON (id) id, version, metadata, encrypted_dek, status, storage_type, 
					   created_at, updated_at, revoked_at
				FROM keys 
				ORDER BY id, version DESC
			)
			SELECT id, version, metadata, encrypted_dek, status, storage_type, 
				   created_at, updated_at, revoked_at 
			FROM latest_keys
			ORDER BY created_at DESC`,
	}

	for name, sql := range statements {
		_, err := conn.Conn().Prepare(ctx, name, sql)
		if err != nil {
			return fmt.Errorf("failed to prepare statement %s: %w", name, err)
		}
	}

	return nil
}

func (s *NeonDBStorage) ensurePrepared(ctx context.Context) {
	s.prepareMux.Do(func() {
		if err := s.prepareStatements(ctx); err != nil {
			s.logger.Error("failed to prepare statements", "error", err)
		} else {
			s.prepared = true
		}
	})
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

	s.ensurePrepared(ctx)

	var key domain.Key
	var metadataRaw []byte
	var storageType string

	var err error
	if s.prepared {
		err = s.db.QueryRow(ctx, stmtGetLatestKey, id.String()).Scan(
			&key.Version, &metadataRaw, &key.EncryptedDEK,
			&key.Status, &storageType, &key.CreatedAt, &key.UpdatedAt, &key.RevokedAt)
	} else {
		const query = `SELECT version, metadata, encrypted_dek, status, storage_type, created_at, updated_at, revoked_at 
		               FROM keys WHERE id = $1::uuid ORDER BY version DESC LIMIT 1`
		err = s.db.QueryRow(ctx, query, id.String()).Scan(
			&key.Version, &metadataRaw, &key.EncryptedDEK,
			&key.Status, &storageType, &key.CreatedAt, &key.UpdatedAt, &key.RevokedAt)
	}

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
	if version <= 0 {
		return nil, ErrInvalidVersion
	}

	cacheKey := s.getCacheKey(id, version)
	if key, found := s.cache.Get(ctx, cacheKey); found {
		return key, nil
	}

	s.ensurePrepared(ctx)

	var key domain.Key
	var metadataRaw []byte
	var storageType string

	var err error
	if s.prepared {
		err = s.db.QueryRow(ctx, stmtGetKeyByVersion, id.String(), version).Scan(
			&metadataRaw, &key.EncryptedDEK, &key.Status, &storageType,
			&key.CreatedAt, &key.UpdatedAt, &key.RevokedAt)
	} else {
		const query = `SELECT metadata, encrypted_dek, status, storage_type, created_at, updated_at, revoked_at 
		               FROM keys WHERE id = $1::uuid AND version = $2`
		err = s.db.QueryRow(ctx, query, id.String(), version).Scan(
			&metadataRaw, &key.EncryptedDEK, &key.Status, &storageType,
			&key.CreatedAt, &key.UpdatedAt, &key.RevokedAt)
	}

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

func (s *NeonDBStorage) CreateKey(ctx context.Context, key *domain.Key) error {
	if key == nil {
		return errors.New("key cannot be nil")
	}

	// Check if key already exists
	exists, err := s.Exists(ctx, key.ID)
	if err != nil {
		return fmt.Errorf("failed to check key existence: %w", err)
	}
	if exists {
		return ErrKeyAlreadyExists
	}

	metadataRaw, err := json.Marshal(key.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	storageType := s.getStorageType(key.Metadata.GetStorageType())

	s.ensurePrepared(ctx)

	if s.prepared {
		_, err = s.db.Exec(ctx, stmtCreateKey,
			key.ID.String(), key.Version, metadataRaw, key.EncryptedDEK,
			key.Status, storageType, key.CreatedAt, key.UpdatedAt)
	} else {
		const query = `INSERT INTO keys (id, version, metadata, encrypted_dek, status, storage_type, created_at, updated_at) 
		               VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
		_, err = s.db.Exec(ctx, query,
			key.ID.String(), key.Version, metadataRaw, key.EncryptedDEK,
			key.Status, storageType, key.CreatedAt, key.UpdatedAt)
	}

	if err != nil {
		return fmt.Errorf("failed to create key %s: %w", key.ID.String(), err)
	}

	return nil
}

func (s *NeonDBStorage) ListKeys(ctx context.Context) ([]*domain.Key, error) {
	s.ensurePrepared(ctx)

	var rows pgx.Rows
	var err error

	if s.prepared {
		rows, err = s.db.Query(ctx, stmtListKeys)
	} else {
		// Fallback to original query if prepared statements aren't available
		const query = `SELECT id, version, metadata, encrypted_dek, status, storage_type, created_at, updated_at, revoked_at FROM keys`
		rows, err = s.db.Query(ctx, query)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query keys: %w", err)
	}
	defer rows.Close()

	keys := make([]*domain.Key, 0, 100)

	for rows.Next() {
		var key domain.Key
		var metadataRaw []byte
		var idStr string
		var storageType string

		err := rows.Scan(&idStr, &key.Version, &metadataRaw, &key.EncryptedDEK,
			&key.Status, &storageType, &key.CreatedAt, &key.UpdatedAt, &key.RevokedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan key row: %w", err)
		}

		key.ID, err = domain.KeyIDFromString(idStr)
		if err != nil {
			s.logger.Error("invalid key ID found in database", "keyID", idStr, "error", err)
			continue // Skip invalid keys instead of failing entirely
		}

		if err := json.Unmarshal(metadataRaw, &key.Metadata); err != nil {
			s.logger.Error("failed to unmarshal metadata", "keyID", idStr, "error", err)
			continue // Skip keys with invalid metadata
		}

		keys = append(keys, &key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return keys, nil
}

func (s *NeonDBStorage) UpdateKeyMetadata(ctx context.Context, id domain.KeyID, metadata *pk.KeyMetadata) error {
	if metadata == nil {
		return errors.New("metadata cannot be nil")
	}

	metadataRaw, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	s.ensurePrepared(ctx)

	var result pgconn.CommandTag
	if s.prepared {
		result, err = s.db.Exec(ctx, stmtUpdateMetadata, metadataRaw, time.Now(), id.String())
	} else {
		const query = `UPDATE keys SET metadata = $1, updated_at = $2 WHERE id = $3::uuid AND version = (
			SELECT MAX(version) FROM keys WHERE id = $3::uuid
		)`
		result, err = s.db.Exec(ctx, query, metadataRaw, time.Now(), id.String())
	}

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
	if len(newEncryptedDEK) == 0 {
		return nil, errors.New("new encrypted DEK cannot be empty")
	}

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
	var storageType string

	const selectQuery = `SELECT version, metadata, storage_type FROM keys 
	                     WHERE id = $1::uuid ORDER BY version DESC LIMIT 1 FOR UPDATE`

	err = tx.QueryRow(ctx, selectQuery, id.String()).Scan(&currentVersion, &metadataRaw, &storageType)
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

	const updateQuery = `UPDATE keys SET status = $1 WHERE id = $2::uuid AND version = $3`
	_, err = tx.Exec(ctx, updateQuery, domain.KeyStatusRotated, id.String(), currentVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to update old key version status: %w", err)
	}

	newVersion := currentVersion + 1
	now := time.Now()
	metadata.Version = newVersion
	metadata.UpdatedAt = timestamppb.New(now)

	newMetadataRaw, err := json.Marshal(&metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated metadata: %w", err)
	}

	const insertQuery = `INSERT INTO keys (id, version, metadata, encrypted_dek, status, storage_type, created_at, updated_at) 
	                     VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err = tx.Exec(ctx, insertQuery,
		id.String(), newVersion, newMetadataRaw, newEncryptedDEK,
		domain.KeyStatusActive, storageType, now, now)
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
	s.ensurePrepared(ctx)

	var result pgconn.CommandTag
	var err error

	if s.prepared {
		result, err = s.db.Exec(ctx, stmtRevokeKey, domain.KeyStatusRevoked, time.Now(), id.String())
	} else {
		const query = `UPDATE keys SET status = $1, revoked_at = $2 WHERE id = $3::uuid`
		result, err = s.db.Exec(ctx, query, domain.KeyStatusRevoked, time.Now(), id.String())
	}

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
	s.ensurePrepared(ctx)

	var rows pgx.Rows
	var err error

	if s.prepared {
		rows, err = s.db.Query(ctx, stmtGetVersions, id.String())
	} else {
		const query = `SELECT version, metadata, encrypted_dek, status, storage_type, created_at, updated_at, revoked_at 
		               FROM keys WHERE id = $1::uuid ORDER BY version DESC`
		rows, err = s.db.Query(ctx, query, id.String())
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query key versions: %w", err)
	}
	defer rows.Close()

	keys := make([]*domain.Key, 0, 10)

	for rows.Next() {
		var key domain.Key
		var metadataRaw []byte
		var storageType string

		err := rows.Scan(&key.Version, &metadataRaw, &key.EncryptedDEK,
			&key.Status, &storageType, &key.CreatedAt, &key.UpdatedAt, &key.RevokedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan key version row: %w", err)
		}

		key.ID = id

		if err := json.Unmarshal(metadataRaw, &key.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata for key %s version %d: %w", id.String(), key.Version, err)
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

	s.ensurePrepared(ctx)

	var exists bool
	var err error

	if s.prepared {
		err = s.db.QueryRow(ctx, stmtCheckExists, id.String()).Scan(&exists)
	} else {
		const query = `SELECT EXISTS(SELECT 1 FROM keys WHERE id = $1::uuid LIMIT 1)`
		err = s.db.QueryRow(ctx, query, id.String()).Scan(&exists)
	}

	if err != nil {
		return false, fmt.Errorf("failed to check key existence %s: %w", id.String(), err)
	}

	return exists, nil
}

func (s *NeonDBStorage) Close() error {
	if c, ok := s.cache.(interface{ Stop() }); ok {
		c.Stop()
	}
	s.db.Close()
	return nil
}

// Helper methods

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

func (s *NeonDBStorage) getStorageType(storageProfile pk.StorageProfile) string {
	switch storageProfile {
	case pk.StorageProfile_STORAGE_PROFILE_STANDARD:
		return constants.StorageTypeStandard
	case pk.StorageProfile_STORAGE_PROFILE_HARDENED:
		return constants.StorageTypeHardened
	default:
		return constants.StorageTypeUnknown
	}
}