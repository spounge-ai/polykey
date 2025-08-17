package persistence

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/spounge-ai/polykey/internal/domain"
)

// ScanKeyRow scans a single row from a pgx.Row and returns a domain.Key, excluding the ID.
// This is used for queries where the ID is already known.
func ScanKeyRow(row pgx.Row) (*domain.Key, error) {
	var key domain.Key
	var metadataRaw []byte
	var storageType string

	err := row.Scan(
		&key.Version,
		&metadataRaw,
		&key.EncryptedDEK,
		&key.Status,
		&storageType,
		&key.CreatedAt,
		&key.UpdatedAt,
		&key.RevokedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan key row: %w", err)
	}

	if err := json.Unmarshal(metadataRaw, &key.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &key, nil
}

// ScanKeyRowWithID scans a single row from a pgx.Row and returns a domain.Key, including the ID.
// This is used for queries like ListKeys where the ID is part of the result set.
func ScanKeyRowWithID(row pgx.Row) (*domain.Key, error) {
	var key domain.Key
	var id uuid.UUID
	var metadataRaw []byte
	var storageType string

	err := row.Scan(
		&id,
		&key.Version,
		&metadataRaw,
		&key.EncryptedDEK,
		&key.Status,
		&storageType,
		&key.CreatedAt,
		&key.UpdatedAt,
		&key.RevokedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan key row: %w", err)
	}

	key.ID, err = domain.KeyIDFromString(id.String())
	if err != nil {
		return nil, fmt.Errorf("failed to create key id from uuid string: %w", err)
	}

	if err := json.Unmarshal(metadataRaw, &key.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata for key %s: %w", key.ID.String(), err)
	}

	return &key, nil
}
