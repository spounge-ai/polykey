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

-- Core indexes
CREATE INDEX IF NOT EXISTS idx_keys_latest_version ON keys(id, version DESC);
CREATE INDEX IF NOT EXISTS idx_keys_active_latest ON keys(id, version DESC) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_keys_status_created_at ON keys(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_keys_status ON keys(status);
CREATE INDEX IF NOT EXISTS idx_keys_metadata_gin ON keys USING gin(metadata);
