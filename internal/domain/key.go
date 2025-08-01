package domain

import (
	"context"

	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// Key represents the core domain entity for a key.
type Key struct {
	ID          string
	Version     int32
	Metadata    *pk.KeyMetadata
	EncryptedDEK []byte
}

// KeyRepository defines the interface for storing and retrieving keys.
type KeyRepository interface {
	GetKey(ctx context.Context, id string) (*Key, error)
	CreateKey(ctx context.Context, key *Key) error
}
