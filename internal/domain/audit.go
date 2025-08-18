package domain

import (
	"context"
	"time"
)

type AuditLogger interface {
	AuditLog(ctx context.Context, clientIdentity, operation, keyID, authDecisionID string, success bool, err error)
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
	CreateAuditEventsBatch(ctx context.Context, events []*AuditEvent) error
	GetAuditHistory(ctx context.Context, keyID string, limit int) ([]*AuditEvent, error)
}