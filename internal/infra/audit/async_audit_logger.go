package audit

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/spounge-ai/polykey/internal/domain"
)

// AsyncAuditLoggerConfig holds the configuration for the asynchronous logger.
type AsyncAuditLoggerConfig struct {
	ChannelBufferSize int
	WorkerCount       int
	BatchSize         int
	BatchTimeout      time.Duration
}

// AsyncAuditLogger provides a non-blocking, asynchronous implementation of the AuditLogger interface.
type AsyncAuditLogger struct {
	logger       *slog.Logger
	auditRepo    domain.AuditRepository
	eventChannel chan *domain.AuditEvent
	waitGroup    sync.WaitGroup
	config       AsyncAuditLoggerConfig
}

// NewAsyncAuditLogger creates a new asynchronous audit logger.
func NewAsyncAuditLogger(logger *slog.Logger, auditRepo domain.AuditRepository, config AsyncAuditLoggerConfig) *AsyncAuditLogger {
	return &AsyncAuditLogger{
		logger:       logger,
		auditRepo:    auditRepo,
		eventChannel: make(chan *domain.AuditEvent, config.ChannelBufferSize),
		config:       config,
	}
}

// Start begins the worker goroutines that process audit events.
func (l *AsyncAuditLogger) Start() {
	l.waitGroup.Add(l.config.WorkerCount)
	for i := 0; i < l.config.WorkerCount; i++ {
		go l.worker()
	}
}

// Stop gracefully shuts down the audit logger, ensuring all queued events are processed.
func (l *AsyncAuditLogger) Stop() {
	l.logger.Info("shutting down audit logger")
	close(l.eventChannel)
	l.waitGroup.Wait()
	l.logger.Info("audit logger shut down successfully")
}

// AuditLog sends an audit event to the queue for asynchronous processing.
func (l *AsyncAuditLogger) AuditLog(ctx context.Context, clientIdentity, operation, keyID, authDecisionID string, success bool, err error) {
	// This part of the function remains synchronous to capture the event details immediately.
	event := &domain.AuditEvent{
		ID:             uuid.New().String(),
		ClientIdentity: clientIdentity,
		Operation:      operation,
		KeyID:          keyID,
		AuthDecisionID: authDecisionID,
		Success:        success,
		Timestamp:      time.Now().UTC(),
	}
	if err != nil {
		event.Error = err.Error()
	}

	// The database write is decoupled by sending the event to a channel.
	select {
	case l.eventChannel <- event:
		// Event successfully queued.
	default:
		// This case prevents blocking if the channel is full.
		l.logger.Warn("audit event channel is full, event dropped", "operation", operation, "keyID", keyID)
	}
}

// worker is a background goroutine that reads events from the channel and writes them to the database in batches.
func (l *AsyncAuditLogger) worker() {
	defer l.waitGroup.Done()

	ticker := time.NewTicker(l.config.BatchTimeout)
	defer ticker.Stop()

	batch := make([]*domain.AuditEvent, 0, l.config.BatchSize)

	for {
		select {
		case event, ok := <-l.eventChannel:
			if !ok {
				// Channel is closed, write any remaining events and exit.
				if len(batch) > 0 {
					l.writeBatchToDB(batch)
				}
				return
			}
			batch = append(batch, event)
			if len(batch) >= l.config.BatchSize {
				l.writeBatchToDB(batch)
				batch = make([]*domain.AuditEvent, 0, l.config.BatchSize) // Reset batch
			}
		case <-ticker.C:
			// Timeout reached, write any events in the current batch.
			if len(batch) > 0 {
				l.writeBatchToDB(batch)
				batch = make([]*domain.AuditEvent, 0, l.config.BatchSize) // Reset batch
			}
		}
	}
}

// writeBatchToDB writes a batch of audit events to the database.
// Note: This implementation does not yet include the dead-letter queue logic for simplicity in this step.
func (l *AsyncAuditLogger) writeBatchToDB(batch []*domain.AuditEvent) {
	if len(batch) == 0 {
		return
	}

	// In a real implementation, you would use pgx.Batch here for efficiency.
	// For simplicity in this step, we will insert one by one.
	for _, event := range batch {
		if err := l.auditRepo.CreateAuditEvent(context.Background(), event); err != nil {
			l.logger.Error("failed to write audit event to database", "error", err, "audit_id", event.ID)
			// Here you would add retry logic and dead-letter queue handling.
		}
	}
}
