package persistence

import (
	"context"
	"encoding/json"

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
	return nil, nil // not implemented
}

func (s *NeonDBStorage) UpdateKeyMetadata(ctx context.Context, id string, metadata *pk.KeyMetadata) error {
	return nil // not implemented
}

func (s *NeonDBStorage) RotateKey(ctx context.Context, id string, newEncryptedDEK []byte) (*domain.Key, error) {
	return nil, nil // not implemented
}

func (s *NeonDBStorage) RevokeKey(ctx context.Context, id string) error {
	return nil // not implemented
}

func (s *NeonDBStorage) GetKeyVersions(ctx context.Context, id string) ([]*domain.Key, error) {
	return nil, nil // not implemented
}