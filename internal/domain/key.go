package domain

import (
	"context"
	"time"

	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// Key represents the core domain entity for a key.
type Key struct {
	ID           string
	Version      int32
	Metadata     *pk.KeyMetadata
	EncryptedDEK []byte
	Status       KeyStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
	RevokedAt    *time.Time
}

// KeyStatus represents the status of a key.
type KeyStatus string

const (
	KeyStatusActive   KeyStatus = "active"
	KeyStatusRotated  KeyStatus = "rotated"
	KeyStatusRevoked  KeyStatus = "revoked"
)

// KeyRepository defines the interface for storing and retrieving keys.
type KeyRepository interface {
	GetKey(ctx context.Context, id string) (*Key, error)
	GetKeyByVersion(ctx context.Context, id string, version int32) (*Key, error)
	CreateKey(ctx context.Context, key *Key) error
	ListKeys(ctx context.Context) ([]*Key, error)
	UpdateKeyMetadata(ctx context.Context, id string, metadata *pk.KeyMetadata) error
	RotateKey(ctx context.Context, id string, newEncryptedDEK []byte) (*Key, error)
	RevokeKey(ctx context.Context, id string) error
	GetKeyVersions(ctx context.Context, id string) ([]*Key, error)
}

// AuditEvent represents an audit event.
type AuditEvent struct {
	ID               string
	ClientIdentity   string
	Operation        string
	KeyID            string
	AuthDecisionID   string
	Success          bool
	Error            string
	Timestamp        time.Time
	RequestMetadata  map[string]string
}

// AuditRepository defines the interface for storing audit events.
type AuditRepository interface {
	CreateAuditEvent(ctx context.Context, event *AuditEvent) error
	GetAuditHistory(ctx context.Context, keyID string, limit int) ([]*AuditEvent, error)
}