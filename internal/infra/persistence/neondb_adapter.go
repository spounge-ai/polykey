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
	consts "github.com/spounge-ai/polykey/internal/constants"
	"github.com/spounge-ai/polykey/internal/domain"
	app_errors "github.com/spounge-ai/polykey/internal/errors"
	"github.com/spounge-ai/polykey/pkg/cache"
	psql "github.com/spounge-ai/polykey/pkg/postgres"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type NeonDBAdapter struct {
	*PostgresBase
	cache            cache.Store[string, *domain.Key]
	cacheIndex       map[string]map[string]struct{}
	cacheIndexMux    sync.RWMutex
	queryBuilderPool *sync.Pool
}

func NewNeonDBAdapter(db *pgxpool.Pool, logger *slog.Logger) (*NeonDBAdapter, error) {
	a := &NeonDBAdapter{
		PostgresBase: NewPostgresBase(db, logger),
		cacheIndex:   make(map[string]map[string]struct{}),
		queryBuilderPool: &sync.Pool{
			New: func() interface{} {
				return &strings.Builder{}
			},
		},
	}

	c := cache.New[string, *domain.Key](
		cache.WithDefaultTTL[string, *domain.Key](5*time.Minute),
		cache.WithCleanupInterval[string, *domain.Key](10*time.Minute),
		cache.WithEvictionCallback[string, *domain.Key](a.onCacheEvict),
	)

	a.cache = c

	if err := a.PrepareStatements(context.Background(), consts.Queries); err != nil {
		return nil, fmt.Errorf("failed to prepare statements: %w", err)
	}

	return a, nil
}

func (a *NeonDBAdapter) onCacheEvict(cacheKey string, key *domain.Key) {
	if key == nil {
		return
	}
	a.cacheIndexMux.Lock()
	defer a.cacheIndexMux.Unlock()

	keyIDStr := key.ID.String()
	if keys, ok := a.cacheIndex[keyIDStr]; ok {
		delete(keys, cacheKey)
		if len(keys) == 0 {
			delete(a.cacheIndex, keyIDStr)
		}
	}
}

func (a *NeonDBAdapter) GetKey(ctx context.Context, id domain.KeyID) (*domain.Key, error) {
	cacheKey := a.getCacheKey(id, 0)
	if key, found := a.cache.Get(ctx, cacheKey); found {
		return key, nil
	}

	row := a.DB.QueryRow(ctx, consts.StmtGetLatestKey, id.String())
	key, err := psql.ScanKeyRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, psql.ErrKeyNotFound
		}
		return nil, err
	}

	key.ID = id
	a.storeInCache(ctx, cacheKey, key)

	return key, nil
}

func (a *NeonDBAdapter) GetKeyByVersion(ctx context.Context, id domain.KeyID, version int32) (*domain.Key, error) {
	if version <= 0 {
		return nil, psql.ErrInvalidVersion
	}

	cacheKey := a.getCacheKey(id, version)
	if key, found := a.cache.Get(ctx, cacheKey); found {
		return key, nil
	}

	row := a.DB.QueryRow(ctx, consts.StmtGetKeyByVersion, id.String(), version)
	key, err := psql.ScanKeyRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, psql.ErrKeyNotFound
		}
		return nil, err
	}

	key.ID = id
	key.Version = version
	a.storeInCache(ctx, cacheKey, key)

	return key, nil
}

func (a *NeonDBAdapter) CreateKey(ctx context.Context, key *domain.Key) error {
	if key == nil {
		return errors.New("key cannot be nil")
	}

	// Check if key already exists
	exists, err := a.Exists(ctx, key.ID)
	if err != nil {
		return fmt.Errorf("failed to check key existence: %w", err)
	}
	if exists {
		return psql.ErrKeyAlreadyExists
	}

	metadataRaw, err := json.Marshal(key.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	storageType := a.getStorageType(key.Metadata.GetStorageType())

	_, err = a.DB.Exec(ctx, consts.StmtCreateKey,
		key.ID.String(), key.Version, metadataRaw, key.EncryptedDEK,
		key.Status, storageType, key.CreatedAt, key.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create key %s: %w", key.ID.String(), err)
	}

	return nil
}

func (a *NeonDBAdapter) ListKeys(ctx context.Context) ([]*domain.Key, error) {
	rows, err := a.DB.Query(ctx, consts.StmtListKeys)
	if err != nil {
		return nil, fmt.Errorf("failed to query keys: %w", err)
	}
	defer rows.Close()

	keys := make([]*domain.Key, 0, 100)
	for rows.Next() {
		key, err := psql.ScanKeyRowWithID(rows)
		if err != nil {
			a.logger.Error("failed to scan key row in ListKeys", "error", err)
			continue
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return keys, nil
}

func (a *NeonDBAdapter) UpdateKeyMetadata(ctx context.Context, id domain.KeyID, metadata *pk.KeyMetadata) error {
	if metadata == nil {
		return errors.New("metadata cannot be nil")
	}

	metadataRaw, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	result, err := a.DB.Exec(ctx, consts.StmtUpdateMetadata, metadataRaw, time.Now(), id.String())

	if err != nil {
		return fmt.Errorf("failed to update key metadata %s: %w", id.String(), err)
	}

	if result.RowsAffected() == 0 {
		return psql.ErrKeyNotFound
	}

	a.invalidateCache(ctx, id)
	return nil
}

func (a *NeonDBAdapter) RotateKey(ctx context.Context, id domain.KeyID, newEncryptedDEK []byte) (*domain.Key, error) {
	if len(newEncryptedDEK) == 0 {
		return nil, errors.New("new encrypted DEK cannot be empty")
	}

	tx, err := a.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if rErr := tx.Rollback(ctx); rErr != nil &&
			!errors.Is(rErr, context.Canceled) &&
			!errors.Is(rErr, pgx.ErrTxClosed) {
			a.logger.Error("failed to rollback transaction", "error", rErr)
		}
	}()

	lockID := a.GetLockID(id)
	locked, err := a.TryAcquireLock(ctx, tx, lockID)
	if err != nil {
		return nil, err
	}
	if !locked {
		return nil, app_errors.ErrKeyRotationLocked
	}

	var currentVersion int32
	var metadataRaw []byte
	var storageType string

	const selectQuery = `SELECT version, metadata, storage_type FROM keys 
	                     WHERE id = $1::uuid ORDER BY version DESC LIMIT 1 FOR UPDATE`

	err = tx.QueryRow(ctx, selectQuery, id.String()).Scan(&currentVersion, &metadataRaw, &storageType)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, psql.ErrKeyNotFound
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

	a.invalidateCache(ctx, id)

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

func (a *NeonDBAdapter) RevokeKey(ctx context.Context, id domain.KeyID) error {
	result, err := a.DB.Exec(ctx, consts.StmtRevokeKey, domain.KeyStatusRevoked, time.Now(), id.String())

	if err != nil {
		return fmt.Errorf("failed to revoke key %s: %w", id.String(), err)
	}

	if result.RowsAffected() == 0 {
		return psql.ErrKeyNotFound
	}

	a.invalidateCache(ctx, id)
	return nil
}

func (a *NeonDBAdapter) GetKeyVersions(ctx context.Context, id domain.KeyID) ([]*domain.Key, error) {
	rows, err := a.DB.Query(ctx, consts.StmtGetVersions, id.String())
	if err != nil {
		return nil, fmt.Errorf("failed to query key versions: %w", err)
	}
	defer rows.Close()

	keys := make([]*domain.Key, 0, 10)
	for rows.Next() {
		key, err := psql.ScanKeyRow(rows)
		if err != nil {
			a.logger.Error("failed to scan key row in GetKeyVersions", "error", err)
			continue
		}
		key.ID = id
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over key versions: %w", err)
	}

	return keys, nil
}

func (a *NeonDBAdapter) Exists(ctx context.Context, id domain.KeyID) (bool, error) {
	cacheKey := a.getCacheKey(id, 0)
	if _, ok := a.cache.Get(ctx, cacheKey); ok {
		return true, nil
	}

	var exists bool
	err := a.DB.QueryRow(ctx, consts.StmtCheckExists, id.String()).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("failed to check key existence %s: %w", id.String(), err)
	}

	return exists, nil
}

func (a *NeonDBAdapter) Close() error {
	if c, ok := a.cache.(interface{ Stop() }); ok {
		c.Stop()
	}
	a.DB.Close()
	return nil
}

// Helper methods

func (a *NeonDBAdapter) getCacheKey(id domain.KeyID, version int32) string {
	sb := a.queryBuilderPool.Get().(*strings.Builder)
	defer a.queryBuilderPool.Put(sb)
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

func (a *NeonDBAdapter) storeInCache(ctx context.Context, cacheKey string, k *domain.Key) {
	a.cache.Set(ctx, cacheKey, k, 0) // Use 0 for default TTL

	a.cacheIndexMux.Lock()
	defer a.cacheIndexMux.Unlock()
	keyIDStr := k.ID.String()
	if _, ok := a.cacheIndex[keyIDStr]; !ok {
		a.cacheIndex[keyIDStr] = make(map[string]struct{})
	}
	a.cacheIndex[keyIDStr][cacheKey] = struct{}{}
}

func (a *NeonDBAdapter) invalidateCache(ctx context.Context, id domain.KeyID) {
	a.cacheIndexMux.RLock()
	keyIDStr := id.String()
	keysToDel, ok := a.cacheIndex[keyIDStr]
	a.cacheIndexMux.RUnlock()

	if !ok {
		return
	}

	for cacheKey := range keysToDel {
		a.cache.Delete(ctx, cacheKey)
	}
}

func (a *NeonDBAdapter) getStorageType(storageProfile pk.StorageProfile) string {
	switch storageProfile {
	case pk.StorageProfile_STORAGE_PROFILE_STANDARD:
		return consts.StorageTypeStandard
	case pk.StorageProfile_STORAGE_PROFILE_HARDENED:
		return consts.StorageTypeHardened
	default:
		return consts.StorageTypeUnknown
	}
}