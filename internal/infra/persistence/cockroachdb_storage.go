package persistence

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spounge-ai/polykey/internal/domain"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

type CockroachDBStorage struct {
	db *pgxpool.Pool
}

func NewCockroachDBStorage(db *pgxpool.Pool) (*CockroachDBStorage, error) {
	return &CockroachDBStorage{db: db}, nil
}

func (s *CockroachDBStorage) GetKey(ctx context.Context, id string) (*domain.Key, error) {
	query := `SELECT version, key_type, status, created_at, updated_at, expires_at, last_accessed_at, creator_identity, authorized_contexts, access_policies, description, tags, data_classification, metadata_checksum, access_count, encrypted_dek, revoked_at FROM keys WHERE id = $1 ORDER BY version DESC LIMIT 1`
	row := s.db.QueryRow(ctx, query, id)

	var key domain.Key
	key.Metadata = &pk.KeyMetadata{}
	var accessPoliciesRaw, tagsRaw []byte

	err := row.Scan(&key.Version, &key.Metadata.KeyType, &key.Metadata.Status, &key.Metadata.CreatedAt, &key.Metadata.UpdatedAt, &key.Metadata.ExpiresAt, &key.Metadata.LastAccessedAt, &key.Metadata.CreatorIdentity, &key.Metadata.AuthorizedContexts, &accessPoliciesRaw, &key.Metadata.Description, &tagsRaw, &key.Metadata.DataClassification, &key.Metadata.MetadataChecksum, &key.Metadata.AccessCount, &key.EncryptedDEK, &key.RevokedAt)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(accessPoliciesRaw, &key.Metadata.AccessPolicies); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(tagsRaw, &key.Metadata.Tags); err != nil {
		return nil, err
	}

	key.ID = id
	return &key, nil
}

func (s *CockroachDBStorage) GetKeyByVersion(ctx context.Context, id string, version int32) (*domain.Key, error) {
	query := `SELECT key_type, status, created_at, updated_at, expires_at, last_accessed_at, creator_identity, authorized_contexts, access_policies, description, tags, data_classification, metadata_checksum, access_count, encrypted_dek, revoked_at FROM keys WHERE id = $1 AND version = $2`
	row := s.db.QueryRow(ctx, query, id, version)

	var key domain.Key
	key.Metadata = &pk.KeyMetadata{}
	var accessPoliciesRaw, tagsRaw []byte

	err := row.Scan(&key.Metadata.KeyType, &key.Metadata.Status, &key.Metadata.CreatedAt, &key.Metadata.UpdatedAt, &key.Metadata.ExpiresAt, &key.Metadata.LastAccessedAt, &key.Metadata.CreatorIdentity, &key.Metadata.AuthorizedContexts, &accessPoliciesRaw, &key.Metadata.Description, &tagsRaw, &key.Metadata.DataClassification, &key.Metadata.MetadataChecksum, &key.Metadata.AccessCount, &key.EncryptedDEK, &key.RevokedAt)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(accessPoliciesRaw, &key.Metadata.AccessPolicies); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(tagsRaw, &key.Metadata.Tags); err != nil {
		return nil, err
	}

	key.ID = id
	key.Version = version
	return &key, nil
}

func (s *CockroachDBStorage) CreateKey(ctx context.Context, key *domain.Key, isPremium bool) error {
	accessPoliciesRaw, err := json.Marshal(key.Metadata.AccessPolicies)
	if err != nil {
		return err
	}

	tagsRaw, err := json.Marshal(key.Metadata.Tags)
	if err != nil {
		return err
	}

	query := `INSERT INTO keys (id, version, key_type, status, created_at, updated_at, expires_at, last_accessed_at, creator_identity, authorized_contexts, access_policies, description, tags, data_classification, metadata_checksum, access_count, encrypted_dek, revoked_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`
	_, err = s.db.Exec(ctx, query, key.ID, key.Version, key.Metadata.KeyType, key.Metadata.Status, key.Metadata.CreatedAt, key.Metadata.UpdatedAt, key.Metadata.ExpiresAt, key.Metadata.LastAccessedAt, key.Metadata.CreatorIdentity, key.Metadata.AuthorizedContexts, accessPoliciesRaw, key.Metadata.Description, tagsRaw, key.Metadata.DataClassification, key.Metadata.MetadataChecksum, key.Metadata.AccessCount, key.EncryptedDEK, key.RevokedAt)
	return err
}

func (s *CockroachDBStorage) ListKeys(ctx context.Context) ([]*domain.Key, error) {
	query := `SELECT id, version, key_type, status, created_at, updated_at, expires_at, last_accessed_at, creator_identity, authorized_contexts, access_policies, description, tags, data_classification, metadata_checksum, access_count, encrypted_dek, revoked_at FROM keys`
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*domain.Key
	for rows.Next() {
		var key domain.Key
		key.Metadata = &pk.KeyMetadata{}
		var accessPoliciesRaw, tagsRaw []byte

		err := rows.Scan(&key.ID, &key.Version, &key.Metadata.KeyType, &key.Metadata.Status, &key.Metadata.CreatedAt, &key.Metadata.UpdatedAt, &key.Metadata.ExpiresAt, &key.Metadata.LastAccessedAt, &key.Metadata.CreatorIdentity, &key.Metadata.AuthorizedContexts, &accessPoliciesRaw, &key.Metadata.Description, &tagsRaw, &key.Metadata.DataClassification, &key.Metadata.MetadataChecksum, &key.Metadata.AccessCount, &key.EncryptedDEK, &key.RevokedAt)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(accessPoliciesRaw, &key.Metadata.AccessPolicies); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(tagsRaw, &key.Metadata.Tags); err != nil {
			return nil, err
		}

		keys = append(keys, &key)
	}

	return keys, nil
}

func (s *CockroachDBStorage) UpdateKeyMetadata(ctx context.Context, id string, metadata *pk.KeyMetadata) error {
	accessPoliciesRaw, err := json.Marshal(metadata.AccessPolicies)
	if err != nil {
		return err
	}

	tagsRaw, err := json.Marshal(metadata.Tags)
	if err != nil {
		return err
	}

	query := `UPDATE keys SET key_type = $1, status = $2, updated_at = $3, expires_at = $4, last_accessed_at = $5, creator_identity = $6, authorized_contexts = $7, access_policies = $8, description = $9, tags = $10, data_classification = $11, metadata_checksum = $12, access_count = $13 WHERE id = $14 AND version = $15`
	_, err = s.db.Exec(ctx, query, metadata.KeyType, metadata.Status, metadata.UpdatedAt, metadata.ExpiresAt, metadata.LastAccessedAt, metadata.CreatorIdentity, metadata.AuthorizedContexts, accessPoliciesRaw, metadata.Description, tagsRaw, metadata.DataClassification, metadata.MetadataChecksum, metadata.AccessCount, id, metadata.Version)
	return err
}

func (s *CockroachDBStorage) RotateKey(ctx context.Context, id string, newEncryptedDEK []byte) (*domain.Key, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			log.Printf("failed to rollback transaction: %v", err)
		}
	}()

	key, err := s.GetKey(ctx, id)
	if err != nil {
		return nil, err
	}

	key.Version++
	key.EncryptedDEK = newEncryptedDEK
	key.UpdatedAt = time.Now()

	if err := s.CreateKey(ctx, key, key.GetTier() == domain.TierPro || key.GetTier() == domain.TierEnterprise); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return key, nil
}

func (s *CockroachDBStorage) RevokeKey(ctx context.Context, id string) error {
	query := `UPDATE keys SET status = $1, revoked_at = $2 WHERE id = $3`
	_, err := s.db.Exec(ctx, query, domain.KeyStatusRevoked, time.Now(), id)
	return err
}

func (s *CockroachDBStorage) GetKeyVersions(ctx context.Context, id string) ([]*domain.Key, error) {
	query := `SELECT version, key_type, status, created_at, updated_at, expires_at, last_accessed_at, creator_identity, authorized_contexts, access_policies, description, tags, data_classification, metadata_checksum, access_count, encrypted_dek, revoked_at FROM keys WHERE id = $1 ORDER BY version DESC`
	rows, err := s.db.Query(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*domain.Key
	for rows.Next() {
		var key domain.Key
		key.Metadata = &pk.KeyMetadata{}
		var accessPoliciesRaw, tagsRaw []byte

		err := rows.Scan(&key.Version, &key.Metadata.KeyType, &key.Metadata.Status, &key.Metadata.CreatedAt, &key.Metadata.UpdatedAt, &key.Metadata.ExpiresAt, &key.Metadata.LastAccessedAt, &key.Metadata.CreatorIdentity, &key.Metadata.AuthorizedContexts, &accessPoliciesRaw, &key.Metadata.Description, &tagsRaw, &key.Metadata.DataClassification, &key.Metadata.MetadataChecksum, &key.Metadata.AccessCount, &key.EncryptedDEK, &key.RevokedAt)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(accessPoliciesRaw, &key.Metadata.AccessPolicies); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(tagsRaw, &key.Metadata.Tags); err != nil {
			return nil, err
		}

		keys = append(keys, &key)
	}

	return keys, nil
}