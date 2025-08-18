package domain

import (
	"context"
	"time"

	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

type Key struct {
    ID           KeyID
    Version      int32
    Metadata     *pk.KeyMetadata
    EncryptedDEK []byte
    Status       KeyStatus
    Tier         KeyTier     
    CreatedAt    time.Time
    UpdatedAt    time.Time
    RevokedAt    *time.Time
}

type KeyTier string

const (
	TierFree       KeyTier = "free"
	TierPro        KeyTier = "pro"
	TierEnterprise KeyTier = "enterprise"
	TierUnknown    KeyTier = "unknown"
)

type KeyStatus string

const (
	KeyStatusActive   KeyStatus = "active"
	KeyStatusRotated  KeyStatus = "rotated"
	KeyStatusRevoked  KeyStatus = "revoked"
)


type KeyRepository interface {
	GetKey(ctx context.Context, id KeyID) (*Key, error)
	GetKeyByVersion(ctx context.Context, id KeyID, version int32) (*Key, error)
	GetKeyMetadata(ctx context.Context, id KeyID) (*pk.KeyMetadata, error)
	GetKeyMetadataByVersion(ctx context.Context, id KeyID, version int32) (*pk.KeyMetadata, error)
	CreateKey(ctx context.Context, key *Key) (*Key, error)
	CreateBatchKeys(ctx context.Context, keys []*Key) error
	ListKeys(ctx context.Context, lastCreatedAt *time.Time, limit int) ([]*Key, error)
	UpdateKeyMetadata(ctx context.Context, id KeyID, metadata *pk.KeyMetadata) error
	RotateKey(ctx context.Context, id KeyID, newEncryptedDEK []byte) (*Key, error)
	RevokeKey(ctx context.Context, id KeyID) error
	GetKeyVersions(ctx context.Context, id KeyID) ([]*Key, error)
	Exists(ctx context.Context, id KeyID) (bool, error)
	GetBatchKeys(ctx context.Context, ids []KeyID) ([]*Key, error)
	GetBatchKeyMetadata(ctx context.Context, ids []KeyID) ([]*pk.KeyMetadata, error)
	RevokeBatchKeys(ctx context.Context, ids []KeyID) error
	UpdateBatchKeyMetadata(ctx context.Context, updates []*Key) error
}
 