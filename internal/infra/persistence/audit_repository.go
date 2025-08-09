package persistence

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spounge-ai/polykey/internal/domain"
)

type AuditRepository struct {
	db *pgxpool.Pool
}

func NewAuditRepository(db *pgxpool.Pool) (*AuditRepository, error) {
	return &AuditRepository{db: db}, nil
}

func (r *AuditRepository) CreateAuditEvent(ctx context.Context, event *domain.AuditEvent) error {
	query := `INSERT INTO audit_events (id, client_identity, operation, key_id, auth_decision_id, success, error_message, timestamp) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.Exec(ctx, query, event.ID, event.ClientIdentity, event.Operation, event.KeyID, event.AuthDecisionID, event.Success, event.Error, event.Timestamp)
	return err
}

func (r *AuditRepository) GetAuditHistory(ctx context.Context, keyID string, limit int) ([]*domain.AuditEvent, error) {
	query := `SELECT id, client_identity, operation, key_id, auth_decision_id, success, error_message, timestamp FROM audit_events WHERE key_id = $1 ORDER BY timestamp DESC LIMIT $2`
	rows, err := r.db.Query(ctx, query, keyID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*domain.AuditEvent
	for rows.Next() {
		var event domain.AuditEvent
		err := rows.Scan(&event.ID, &event.ClientIdentity, &event.Operation, &event.KeyID, &event.AuthDecisionID, &event.Success, &event.Error, &event.Timestamp)
		if err != nil {
			return nil, err
		}
		events = append(events, &event)
	}

	return events, nil
}
