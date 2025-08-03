package audit

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/spounge-ai/polykey/internal/domain"
)

// Logger implements the domain.AuditLogger interface.
type Logger struct {
	logger   *slog.Logger
	auditRepo domain.AuditRepository
}

// NewAuditLogger creates a new audit logger.
func NewAuditLogger(logger *slog.Logger, auditRepo domain.AuditRepository) domain.AuditLogger {
	return &Logger{
		logger:    logger,
		auditRepo: auditRepo,
	}
}

// AuditLog logs an audit event both to structured logs and persistent storage.
func (l *Logger) AuditLog(ctx context.Context, clientIdentity, operation, keyID, authDecisionID string, success bool, err error) {
	event := &domain.AuditEvent{
		ID:             uuid.New().String(),
		ClientIdentity: clientIdentity,
		Operation:      operation,
		KeyID:          keyID,
		AuthDecisionID: authDecisionID,
		Success:        success,
		Timestamp:      time.Now().UTC(),
		RequestMetadata: map[string]string{
			"user_agent": extractUserAgent(ctx),
			"source_ip":  extractSourceIP(ctx),
		},
	}

	if err != nil {
		event.Error = err.Error()
	}

	// Log to structured logger
	logAttrs := []slog.Attr{
		slog.String("audit_id", event.ID),
		slog.String("client_identity", clientIdentity),
		slog.String("operation", operation),
		slog.String("key_id", keyID),
		slog.String("auth_decision_id", authDecisionID),
		slog.Bool("success", success),
		slog.Time("timestamp", event.Timestamp),
	}

	if err != nil {
		logAttrs = append(logAttrs, slog.String("error", err.Error()))
	}

	l.logger.LogAttrs(ctx, slog.LevelInfo, "audit_event", logAttrs...)

	// Store in audit repository
	if l.auditRepo != nil {
		if auditErr := l.auditRepo.CreateAuditEvent(ctx, event); auditErr != nil {
			l.logger.ErrorContext(ctx, "failed to store audit event", 
				slog.String("audit_id", event.ID),
				slog.String("error", auditErr.Error()))
		}
	}
}

// extractUserAgent extracts user agent from context metadata.
func extractUserAgent(ctx context.Context) string {
	// Implementation depends on how metadata is stored in context
	// This is a placeholder - adjust based on your gRPC metadata handling
	return "unknown"
}

// extractSourceIP extracts source IP from context metadata.
func extractSourceIP(ctx context.Context) string {
	// Implementation depends on how metadata is stored in context
	// This is a placeholder - adjust based on your gRPC metadata handling
	return "unknown"
}