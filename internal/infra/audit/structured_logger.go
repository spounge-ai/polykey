package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spounge-ai/polykey/internal/domain"
)

// Placeholders for dependencies to be implemented later.
type IntegrityManager struct{}
type LogSanitizer struct{}

func NewIntegrityManager() *IntegrityManager {
	return &IntegrityManager{}
}

func NewLogSanitizer() *LogSanitizer {
	return &LogSanitizer{}
}

func (s *LogSanitizer) SanitizeEvent(event *AuditEvent) {}

type StructuredAuditLogger struct {
	logger      *slog.Logger
	repository  domain.AuditRepository
	integrity   *IntegrityManager
	sanitizer   *LogSanitizer
}

func NewStructuredAuditLogger(
	logger *slog.Logger,
	repo domain.AuditRepository,
	integrity *IntegrityManager,
) *StructuredAuditLogger {
	return &StructuredAuditLogger{
		logger:     logger,
		repository: repo,
		integrity:  integrity,
		sanitizer:  NewLogSanitizer(),
	}
}

func (sal *StructuredAuditLogger) LogKeyOperation(ctx context.Context, req *KeyOperationRequest) error {
	event := &AuditEvent{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		EventType: "key_operation",
		Action:    req.Operation,
		Result:    req.Result,
		Details:   make(map[string]interface{}),
	}

	if user, ok := domain.UserFromContext(ctx); ok {
		event.Actor = &ActorInfo{
			UserID:    user.ID,
			Tier:      string(user.Tier),
			SessionID: req.SessionID,
			ClientIP:  req.ClientIP,
			UserAgent: req.UserAgent,
		}
	}

	if req.KeyID != "" {
		event.Resource = &ResourceInfo{
			Type: "cryptographic_key",
			ID:   req.KeyID,
			Classification: req.DataClassification,
		}
		event.SecurityLevel = sal.determineSecurityLevel(req.DataClassification)
	}

	sal.addOperationDetails(event, req)
	sal.sanitizer.SanitizeEvent(event)
	event.Checksum = sal.generateChecksum(event)

	// For now, we will just log to the structured logger.
	// Storing to the repository will be part of a later step.
	sal.logToStructuredLogger(ctx, event)

	return nil
}

func (sal *StructuredAuditLogger) determineSecurityLevel(classification string) string {
	switch strings.ToLower(classification) {
	case "restricted":
		return "high"
	case "confidential":
		return "medium"
	default:
		return "low"
	}
}

func (sal *StructuredAuditLogger) generateChecksum(event *AuditEvent) string {
	data := map[string]interface{}{
		"id":            event.ID,
		"timestamp":     event.Timestamp.Unix(),
		"event_type":    event.EventType,
		"action":        event.Action,
		"result":        event.Result,
		"security_level": event.SecurityLevel,
	}

	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

func (sal *StructuredAuditLogger) addOperationDetails(event *AuditEvent, req *KeyOperationRequest) {
	if req.AdditionalContext != nil {
		for k, v := range req.AdditionalContext {
			event.Details[k] = v
		}
	}
}

func (sal *StructuredAuditLogger) logToStructuredLogger(ctx context.Context, event *AuditEvent) {
	logAttrs := []slog.Attr{
		slog.String("audit_id", event.ID),
		slog.String("event_type", event.EventType),
		slog.String("action", event.Action),
		slog.String("result", event.Result),
		slog.String("security_level", event.SecurityLevel),
		slog.Time("timestamp", event.Timestamp),
	}

	if event.Actor != nil {
		logAttrs = append(logAttrs, slog.Group("actor",
			slog.String("user_id", event.Actor.UserID),
			slog.String("client_ip", event.Actor.ClientIP),
			slog.String("user_agent", event.Actor.UserAgent),
			slog.String("tier", event.Actor.Tier),
		))
	}

	if event.Resource != nil {
		logAttrs = append(logAttrs, slog.Group("resource",
			slog.String("type", event.Resource.Type),
			slog.String("id", event.Resource.ID),
			slog.String("classification", event.Resource.Classification),
		))
	}

	sal.logger.LogAttrs(ctx, slog.LevelInfo, "audit_event", logAttrs...)
}
