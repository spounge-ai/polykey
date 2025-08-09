CREATE TABLE IF NOT EXISTS audit_events (
    id UUID PRIMARY KEY,
    client_identity VARCHAR(255),
    operation VARCHAR(255),
    key_id VARCHAR(255),
    auth_decision_id VARCHAR(255),
    success BOOLEAN,
    error_message TEXT,
    timestamp TIMESTAMPTZ
);