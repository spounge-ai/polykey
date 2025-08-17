CREATE TABLE IF NOT EXISTS audit_events (
    id UUID PRIMARY KEY,
    client_identity VARCHAR(255),
    operation VARCHAR(255),
    key_id VARCHAR(255),
    auth_decision_id VARCHAR(255),
    success BOOLEAN,
    error_message TEXT,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_key_id ON audit_events(key_id);
CREATE INDEX IF NOT EXISTS idx_audit_client_ts ON audit_events(client_identity, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_operation ON audit_events(operation);
CREATE INDEX IF NOT EXISTS idx_audit_auth_decision_id ON audit_events(auth_decision_id);
CREATE INDEX IF NOT EXISTS idx_audit_success_ts ON audit_events(success, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_error_gin ON audit_events USING gin(to_tsvector('english', error_message));
