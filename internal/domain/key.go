package domain

import (
	"context"
	"time"

	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

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

type KeyTier string

const (
	TierFree       KeyTier = "free"
	TierPro        KeyTier = "pro"
	TierEnterprise KeyTier = "enterprise"
	TierUnknown    KeyTier = "unknown"
)

func (k *Key) GetTier() KeyTier {
	if k.Metadata == nil || k.Metadata.Tags == nil {
		return TierFree // Default to free tier
	}
	tier, ok := k.Metadata.Tags["tier"]
	if !ok {
		return TierFree // Default to free tier
	}
	switch tier {
	case "pro":
		return TierPro
	case "enterprise":
		return TierEnterprise
	case "free":
		return TierFree
	default:
		return TierUnknown
	}
}

func (k *Key) IsPremium() bool {
	tier := k.GetTier()
	return tier == TierPro || tier == TierEnterprise
}

type KeyStatus string

const (
	KeyStatusActive   KeyStatus = "active"
	KeyStatusRotated  KeyStatus = "rotated"
	KeyStatusRevoked  KeyStatus = "revoked"
)

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

type AuditRepository interface {
	CreateAuditEvent(ctx context.Context, event *AuditEvent) error
	GetAuditHistory(ctx context.Context, keyID string, limit int) ([]*AuditEvent, error)
}
