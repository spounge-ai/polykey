package persistence

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spounge-ai/polykey/internal/domain"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

type NeonDBStorage struct {
	db *pgxpool.Pool
}

func NewNeonDBStorage(db *pgxpool.Pool) (*NeonDBStorage, error) {
	return &NeonDBStorage{db: db}, nil
}

func (s *NeonDBStorage) GetKey(ctx context.Context, id string) (*domain.Key, error) {
	query := `SELECT version, metadata, encrypted_dek, status, created_at, updated_at, revoked_at FROM keys WHERE id = $1 ORDER BY version DESC LIMIT 1`
	row := s.db.QueryRow(ctx, query, id)

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

func (s *NeonDBStorage) GetKeyByVersion(ctx context.Context, id string, version int32) (*domain.Key, error) {
	query := `SELECT metadata, encrypted_dek, status, created_at, updated_at, revoked_at FROM keys WHERE id = $1 AND version = $2`
	row := s.db.QueryRow(ctx, query, id, version)

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

func (s *NeonDBStorage) CreateKey(ctx context.Context, key *domain.Key) error {
	metadataRaw, err := json.Marshal(key.Metadata)
	if err != nil {
		return err
	}

	query := `INSERT INTO keys (id, version, metadata, encrypted_dek, status, is_premium, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err = s.db.Exec(ctx, query, key.ID, key.Version, metadataRaw, key.EncryptedDEK, key.Status, key.IsPremium(), key.CreatedAt, key.UpdatedAt)
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
		err := rows.Scan(&key.ID, &key.Version, &metadataRaw, &key.EncryptedDEK, &key.Status, &key.CreatedAt, &key.UpdatedAt, &key.RevokedAt)
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

func (s *NeonDBStorage) UpdateKeyMetadata(ctx context.Context, id string, metadata *pk.KeyMetadata) error {
	metadataRaw, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	query := `UPDATE keys SET metadata = $1, updated_at = $2 WHERE id = $3`
	_, err = s.db.Exec(ctx, query, metadataRaw, time.Now(), id)
	return err
}

func (s *NeonDBStorage) RotateKey(ctx context.Context, id string, newEncryptedDEK []byte) (*domain.Key, error) {
	// This is a simplified implementation. A real implementation would use a transaction.
	key, err := s.GetKey(ctx, id)
	if err != nil {
		return nil, err
	}

	key.Version++
	key.EncryptedDEK = newEncryptedDEK
	key.UpdatedAt = time.Now()

	if err := s.CreateKey(ctx, key); err != nil {
		return nil, err
	}

	return key, nil
}

func (s *NeonDBStorage) RevokeKey(ctx context.Context, id string) error {
	query := `UPDATE keys SET status = $1, revoked_at = $2 WHERE id = $3`
	_, err := s.db.Exec(ctx, query, domain.KeyStatusRevoked, time.Now(), id)
	return err
}

func (s *NeonDBStorage) GetKeyVersions(ctx context.Context, id string) ([]*domain.Key, error) {
	query := `SELECT id, version, metadata, encrypted_dek, status, created_at, updated_at, revoked_at FROM keys WHERE id = $1 ORDER BY version DESC`
	rows, err := s.db.Query(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*domain.Key
	for rows.Next() {
		var key domain.Key
		var metadataRaw []byte
		err := rows.Scan(&key.ID, &key.Version, &metadataRaw, &key.EncryptedDEK, &key.Status, &key.CreatedAt, &key.UpdatedAt, &key.RevokedAt)
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
