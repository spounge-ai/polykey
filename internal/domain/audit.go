package domain

import "context"

// AuditLogger defines the interface for audit logging.
type AuditLogger interface {
	AuditLog(ctx context.Context, clientIdentity, operation, keyID, authDecisionID string, success bool, err error)
}
