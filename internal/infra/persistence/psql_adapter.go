package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	consts "github.com/spounge-ai/polykey/internal/constants"
	"github.com/spounge-ai/polykey/internal/domain"
	app_errors "github.com/spounge-ai/polykey/internal/errors"
	psql "github.com/spounge-ai/polykey/pkg/postgres"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	
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

func (a *PSQLAdapter) GetKeyMetadata(ctx context.Context, id domain.KeyID) (*pk.KeyMetadata, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var metadataRaw []byte
	err := a.DB.QueryRow(ctx, consts.Queries[consts.StmtGetKeyMetadata], id.String()).Scan(&metadataRaw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, psql.ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to get key metadata for %s: %w", id.String(), err)
	}

	var metadata pk.KeyMetadata
	if err := json.Unmarshal(metadataRaw, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &metadata, nil
}

func (a *PSQLAdapter) GetKeyMetadataByVersion(ctx context.Context, id domain.KeyID, version int32) (*pk.KeyMetadata, error) {
	if version <= 0 {
		return nil, psql.ErrInvalidVersion
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var metadataRaw []byte
	err := a.DB.QueryRow(ctx, consts.Queries[consts.StmtGetKeyMetadataByVersion], id.String(), version).Scan(&metadataRaw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, psql.ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to get key metadata for %s version %d: %w", id.String(), version, err)
	}

	var metadata pk.KeyMetadata
	if err := json.Unmarshal(metadataRaw, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &metadata, nil
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

	createdKey, err := ScanKeyRowWithID(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return nil, psql.ErrKeyAlreadyExists
		}
		return nil, fmt.Errorf("failed to create key %s: %w", key.ID.String(), err)
	}

	return createdKey, nil
}

func (a *PSQLAdapter) CreateBatchKeys(ctx context.Context, keys []*domain.Key) error {
	if len(keys) == 0 {
		return nil
	}

	columnNames := []string{
		"id", "version", "metadata", "encrypted_dek",
		"status", "storage_type", "created_at", "updated_at",
	}

	rows := make([][]interface{}, len(keys))
	for i, key := range keys {
		metadataRaw, err := a.optimizer.MarshalWithBuffer(key.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata for key %s: %w", key.ID.String(), err)
		}
		rows[i] = []interface{}{
			key.ID.String(), key.Version, metadataRaw, key.EncryptedDEK,
			key.Status, getStorageTypeOptimized(key.Metadata.GetStorageType()), key.CreatedAt, key.UpdatedAt,
		}
	}

	_, err := a.DB.CopyFrom(
		ctx,
		pgx.Identifier{"keys"},
		columnNames,
		pgx.CopyFromRows(rows),
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return psql.ErrKeyAlreadyExists
		}
		return fmt.Errorf("failed to create keys in batch: %w", err)
	}

	return nil
}

func (a *PSQLAdapter) ListKeys(ctx context.Context, lastCreatedAt *time.Time, limit int) ([]*domain.Key, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	rows, err := a.DB.Query(ctx, consts.Queries[consts.StmtListKeys], lastCreatedAt, limit)
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

	const rotateQuery = `
		WITH old_key AS (
			UPDATE keys
			SET status = $1, updated_at = now()
			WHERE id = $2 AND version = (SELECT MAX(version) FROM keys WHERE id = $2)
			RETURNING id, metadata, storage_type
		),
		new_key AS (
			INSERT INTO keys (id, version, metadata, encrypted_dek, status, storage_type, created_at, updated_at)
			SELECT
				id,
				(metadata->>'version')::int + 1,
				jsonb_set(metadata, '{version}', (((metadata->>'version')::int + 1)::text)::jsonb),
				$3,
				$4,
				storage_type,
				now(),
				now()
			FROM old_key
			RETURNING id, version, metadata, encrypted_dek, status, storage_type, created_at, updated_at, revoked_at
		)
		SELECT id, version, metadata, encrypted_dek, status, storage_type, created_at, updated_at, revoked_at FROM new_key;
	`

	row := tx.QueryRow(ctx, rotateQuery,
		domain.KeyStatusRotated,
		id.String(),
		newEncryptedDEK,
		domain.KeyStatusActive,
	)

	key, err := ScanKeyRowWithID(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, psql.ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to rotate key %s: %w", id.String(), err)
	}

	return key, nil
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

