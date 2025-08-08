package domain

import "context"

type AuditLogger interface {
	AuditLog(ctx context.Context, clientIdentity, operation, keyID, authDecisionID string, success bool, err error)
}
