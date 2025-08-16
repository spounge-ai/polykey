DROP TABLE IF EXISTS keys;

CREATE TABLE keys (
    id UUID NOT NULL,
    version INT NOT NULL,
    metadata JSONB NOT NULL,
    encrypted_dek BYTEA NOT NULL,
    status VARCHAR(20) NOT NULL,
    storage_type VARCHAR(20) NOT NULL, 
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    PRIMARY KEY (id, version)
);

CREATE INDEX idx_keys_status ON keys(status);
CREATE INDEX idx_keys_storage_type ON keys(storage_type);
