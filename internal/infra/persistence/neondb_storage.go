package persistence

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spounge-ai/polykey/internal/domain"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type NeonDBStorage struct {
	db *pgxpool.Pool
}

func NewNeonDBStorage(db *pgxpool.Pool) (*NeonDBStorage, error) {
	return &NeonDBStorage{db: db}, nil
}

func (s *NeonDBStorage) GetKey(ctx context.Context, id domain.KeyID) (*domain.Key, error) {
	query := `SELECT version, metadata, encrypted_dek, status, created_at, updated_at, revoked_at FROM keys WHERE id = $1 ORDER BY version DESC LIMIT 1`
	row := s.db.QueryRow(ctx, query, id.String())

	var key domain.Key
	var metadataRaw []byte

	err := row.Scan(&key.Version, &metadataRaw, &key.EncryptedDEK, &key.Status, &key.CreatedAt, &key.UpdatedAt, &key.RevokedAt)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(metadataRaw, &key.Metadata); err != nil {
		return nil, err
	}

	key.ID = id
	return &key, nil
}

func (s *NeonDBStorage) GetKeyByVersion(ctx context.Context, id domain.KeyID, version int32) (*domain.Key, error) {
	query := `SELECT metadata, encrypted_dek, status, created_at, updated_at, revoked_at FROM keys WHERE id = $1 AND version = $2`
	row := s.db.QueryRow(ctx, query, id.String(), version)

	var key domain.Key
	var metadataRaw []byte

	err := row.Scan(&metadataRaw, &key.EncryptedDEK, &key.Status, &key.CreatedAt, &key.UpdatedAt, &key.RevokedAt)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(metadataRaw, &key.Metadata); err != nil {
		return nil, err
	}

	key.ID = id
	key.Version = version
	return &key, nil
}

func (s *NeonDBStorage) CreateKey(ctx context.Context, key *domain.Key, isPremium bool) error {
	metadataRaw, err := json.Marshal(key.Metadata)
	if err != nil {
		return err
	}

	query := `INSERT INTO keys (id, version, metadata, encrypted_dek, status, created_at, updated_at, is_premium) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err = s.db.Exec(ctx, query, key.ID.String(), key.Version, metadataRaw, key.EncryptedDEK, key.Status, key.CreatedAt, key.UpdatedAt, isPremium)
	return err
}

func (s *NeonDBStorage) ListKeys(ctx context.Context) ([]*domain.Key, error) {
	query := `SELECT id, version, metadata, encrypted_dek, status, created_at, updated_at, revoked_at FROM keys`
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*domain.Key
	for rows.Next() {
		var key domain.Key
		var metadataRaw []byte
		var idStr string
		err := rows.Scan(&idStr, &key.Version, &metadataRaw, &key.EncryptedDEK, &key.Status, &key.CreatedAt, &key.UpdatedAt, &key.RevokedAt)
		if err != nil {
			return nil, err
		}

		key.ID, err = domain.KeyIDFromString(idStr)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(metadataRaw, &key.Metadata); err != nil {
			return nil, err
		}

		keys = append(keys, &key)
	}

	return keys, nil
}

func (s *NeonDBStorage) UpdateKeyMetadata(ctx context.Context, id domain.KeyID, metadata *pk.KeyMetadata) error {
	metadataRaw, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	query := `UPDATE keys SET metadata = $1, updated_at = $2 WHERE id = $3`
	_, err = s.db.Exec(ctx, query, metadataRaw, time.Now(), id.String())
	return err
}

func (s *NeonDBStorage) RotateKey(ctx context.Context, id domain.KeyID, newEncryptedDEK []byte) (*domain.Key, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			log.Printf("failed to rollback transaction: %v", err)
		}
	}()

	var currentVersion int32
	var metadataRaw []byte
	var isPremium bool
	query := `SELECT version, metadata, is_premium FROM keys WHERE id = $1 ORDER BY version DESC LIMIT 1`
	err = tx.QueryRow(ctx, query, id.String()).Scan(&currentVersion, &metadataRaw, &isPremium)
	if err != nil {
		return nil, err
	}

	var metadata pk.KeyMetadata
	if err := json.Unmarshal(metadataRaw, &metadata); err != nil {
		return nil, err
	}

	newVersion := currentVersion + 1
	now := time.Now()
	metadata.Version = newVersion
	metadata.UpdatedAt = timestamppb.New(now)

	newMetadataRaw, err := json.Marshal(&metadata)
	if err != nil {
		return nil, err
	}

	insertQuery := `INSERT INTO keys (id, version, metadata, encrypted_dek, status, created_at, updated_at, is_premium) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err = tx.Exec(ctx, insertQuery, id.String(), newVersion, newMetadataRaw, newEncryptedDEK, domain.KeyStatusActive, now, now, isPremium)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
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

func (s *NeonDBStorage) RevokeKey(ctx context.Context, id domain.KeyID) error {
	query := `UPDATE keys SET status = $1, revoked_at = $2 WHERE id = $3`
	_, err := s.db.Exec(ctx, query, domain.KeyStatusRevoked, time.Now(), id.String())
	return err
}

func (s *NeonDBStorage) GetKeyVersions(ctx context.Context, id domain.KeyID) ([]*domain.Key, error) {
	query := `SELECT id, version, metadata, encrypted_dek, status, created_at, updated_at, revoked_at FROM keys WHERE id = $1 ORDER BY version DESC`
	rows, err := s.db.Query(ctx, query, id.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*domain.Key
	for rows.Next() {
		var key domain.Key
		var metadataRaw []byte
		var idStr string
		err := rows.Scan(&idStr, &key.Version, &metadataRaw, &key.EncryptedDEK, &key.Status, &key.CreatedAt, &key.UpdatedAt, &key.RevokedAt)
		if err != nil {
			return nil, err
		}

		key.ID, err = domain.KeyIDFromString(idStr)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(metadataRaw, &key.Metadata); err != nil {
			return nil, err
		}

		keys = append(keys, &key)
	}

	return keys, nil
}
