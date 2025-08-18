package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	consts "github.com/spounge-ai/polykey/internal/constants"
	"github.com/spounge-ai/polykey/internal/domain"
	app_errors "github.com/spounge-ai/polykey/internal/errors"
	psql "github.com/spounge-ai/polykey/pkg/postgres"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultKeysCapacity = 100
	versionsCapacity    = 10
)

type PSQLAdapter struct {
	*PostgresBase
	optimizer *QueryOptimizer
	txManager *TransactionManager[*domain.Key]
}

func NewPSQLAdapter(db *pgxpool.Pool, logger *slog.Logger) (*PSQLAdapter, error) {
	a := &PSQLAdapter{
		PostgresBase: NewPostgresBase(db, logger),
		optimizer:    NewQueryOptimizer(),
		txManager:    NewTransactionManager[*domain.Key](logger),
	}

	return a, nil
}

func (a *PSQLAdapter) GetKey(ctx context.Context, id domain.KeyID) (*domain.Key, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	row := a.DB.QueryRow(ctx, consts.Queries[consts.StmtGetLatestKey], id.String())
	key, err := ScanKeyRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, psql.ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to get key %s: %w", id.String(), err)
	}

	key.ID = id
	return key, nil
}

func (a *PSQLAdapter) GetKeyByVersion(ctx context.Context, id domain.KeyID, version int32) (*domain.Key, error) {
	if version <= 0 {
		return nil, psql.ErrInvalidVersion
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	row := a.DB.QueryRow(ctx, consts.Queries[consts.StmtGetKeyByVersion], id.String(), version)
	key, err := ScanKeyRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, psql.ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to get key %s version %d: %w", id.String(), version, err)
	}

	key.ID = id
	key.Version = version
	return key, nil
}

func (a *PSQLAdapter) CreateKey(ctx context.Context, key *domain.Key) (*domain.Key, error) {
	if key == nil {
		return nil, errors.New("key cannot be nil")
	}
	if key.Metadata == nil {
		return nil, errors.New("key metadata cannot be nil")
	}
	if len(key.EncryptedDEK) == 0 {
		return nil, errors.New("encrypted DEK cannot be empty")
	}

	exists, err := a.Exists(ctx, key.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check key existence: %w", err)
	}
	if exists {
		return nil, psql.ErrKeyAlreadyExists
	}

	metadataRaw, err := a.optimizer.MarshalWithBuffer(key.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	storageType := getStorageTypeOptimized(key.Metadata.GetStorageType())

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	row := a.DB.QueryRow(ctx, consts.Queries[consts.StmtCreateKey],
		key.ID.String(), key.Version, metadataRaw, key.EncryptedDEK,
		key.Status, storageType, key.CreatedAt, key.UpdatedAt)

	if err := row.Scan(&key.Version, &key.CreatedAt, &key.UpdatedAt); err != nil {
		return nil, fmt.Errorf("failed to create key %s: %w", key.ID.String(), err)
	}

	return key, nil
}

func (a *PSQLAdapter) CreateKeys(ctx context.Context, keys []*domain.Key) error {
	batch := &pgx.Batch{}
	for _, key := range keys {
		metadataRaw, err := a.optimizer.MarshalWithBuffer(key.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata for key %s: %w", key.ID.String(), err)
		}
		storageType := getStorageTypeOptimized(key.Metadata.GetStorageType())
		batch.Queue(consts.Queries[consts.StmtCreateKey], key.ID.String(), key.Version, metadataRaw, key.EncryptedDEK, key.Status, storageType, key.CreatedAt, key.UpdatedAt)
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	br := a.DB.SendBatch(ctx, batch)
	defer func() {
		if err := br.Close(); err != nil {
			a.logger.Error("failed to close batch", "error", err)
		}
	}()

	for i := 0; i < len(keys); i++ {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("failed to create key in batch: %w", err)
		}
	}

	return nil
}

func (a *PSQLAdapter) ListKeys(ctx context.Context) ([]*domain.Key, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	rows, err := a.DB.Query(ctx, consts.Queries[consts.StmtListKeys])
	if err != nil {
		return nil, fmt.Errorf("failed to query keys: %w", err)
	}
	defer rows.Close()

	keys := make([]*domain.Key, 0, defaultKeysCapacity)
	for rows.Next() {
		key, err := ScanKeyRowWithID(rows)
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

func (a *PSQLAdapter) UpdateKeyMetadata(ctx context.Context, id domain.KeyID, metadata *pk.KeyMetadata) error {
	if metadata == nil {
		return errors.New("metadata cannot be nil")
	}

	metadataRaw, err := a.optimizer.MarshalWithBuffer(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	result, err := a.DB.Exec(ctx, consts.Queries[consts.StmtUpdateMetadata], metadataRaw, time.Now(), id.String())
	if err != nil {
		return fmt.Errorf("failed to update key metadata %s: %w", id.String(), err)
	}

	if result.RowsAffected() == 0 {
		return psql.ErrKeyNotFound
	}

	return nil
}

func (a *PSQLAdapter) RotateKey(ctx context.Context, id domain.KeyID, newEncryptedDEK []byte) (*domain.Key, error) {
	if len(newEncryptedDEK) == 0 {
		return nil, errors.New("new encrypted DEK cannot be empty")
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return a.txManager.ExecuteInTransaction(ctx, a.DB, func(ctx context.Context, tx pgx.Tx) (*domain.Key, error) {
		return a.rotateKeyInTx(ctx, tx, id, newEncryptedDEK)
	})
}

func (a *PSQLAdapter) rotateKeyInTx(ctx context.Context, tx pgx.Tx, id domain.KeyID, newEncryptedDEK []byte) (*domain.Key, error) {
	lockID := a.GetLockID(id)
	locked, err := a.TryAcquireLock(ctx, tx, lockID)
	if err != nil {
		return nil, err
	}
	if !locked {
		return nil, app_errors.ErrKeyRotationLocked
	}

	const selectQuery = `SELECT version, metadata, storage_type FROM keys 
	                     WHERE id = $1::uuid ORDER BY version DESC LIMIT 1 FOR UPDATE`
	
	var currentVersion int32
	var metadataRaw []byte
	var storageType string

	err = tx.QueryRow(ctx, selectQuery, id.String()).Scan(&currentVersion, &metadataRaw, &storageType)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, psql.ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to get current key version: %w", err)
	}

	const updateQuery = `UPDATE keys SET status = $1 WHERE id = $2::uuid AND version = $3`
	_, err = tx.Exec(ctx, updateQuery, domain.KeyStatusRotated, id.String(), currentVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to update old key version status: %w", err)
	}

	var metadata pk.KeyMetadata
	if err := json.Unmarshal(metadataRaw, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	newVersion := currentVersion + 1
	now := time.Now()
	metadata.Version = newVersion
	metadata.UpdatedAt = timestamppb.New(now)

	newMetadataRaw, err := a.optimizer.MarshalWithBuffer(&metadata)
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

func (a *PSQLAdapter) RevokeKey(ctx context.Context, id domain.KeyID) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	result, err := a.DB.Exec(ctx, consts.Queries[consts.StmtRevokeKey], domain.KeyStatusRevoked, time.Now(), id.String())
	if err != nil {
		return fmt.Errorf("failed to revoke key %s: %w", id.String(), err)
	}

	if result.RowsAffected() == 0 {
		return psql.ErrKeyNotFound
	}

	return nil
}

func (a *PSQLAdapter) GetKeyVersions(ctx context.Context, id domain.KeyID) ([]*domain.Key, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	rows, err := a.DB.Query(ctx, consts.Queries[consts.StmtGetVersions], id.String())
	if err != nil {
		return nil, fmt.Errorf("failed to query key versions: %w", err)
	}
	defer rows.Close()

	keys := make([]*domain.Key, 0, versionsCapacity)
	for rows.Next() {
		key, err := ScanKeyRow(rows)
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

func (a *PSQLAdapter) Exists(ctx context.Context, id domain.KeyID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var exists bool
	err := a.DB.QueryRow(ctx, consts.Queries[consts.StmtCheckExists], id.String()).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check key existence %s: %w", id.String(), err)
	}

	return exists, nil
}

func (a *PSQLAdapter) Close() error {
	a.DB.Close()
	return nil
}

