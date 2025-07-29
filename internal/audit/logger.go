package audit

import (
	"context"
	"log"
)

// Logger defines the interface for audit logging.
type Logger interface {
	AuditLog(ctx context.Context, clientIdentity, operation, keyID, authDecisionID string, success bool, err error)
}

// NewLogger creates a new basic audit logger.
func NewLogger() Logger {
	return &basicLogger{}
}

// basicLogger implements the Logger interface with simplified logging.
type basicLogger struct{}

// AuditLog logs audit events.
func (l *basicLogger) AuditLog(ctx context.Context, clientIdentity, operation, keyID, authDecisionID string, success bool, err error) {
	// In a real system, this would write to a secure, immutable audit log.
	log.Printf("AUDIT: Client=%s, Operation=%s, KeyID=%s, AuthDecision=%s, Success=%t, Error=%v",
		clientIdentity, operation, keyID, authDecisionID, success, err)
}