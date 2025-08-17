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

-- Essential indexes for secrets manager operations
-- 1. Latest version lookup (most common: GetKey by ID)
CREATE INDEX idx_keys_latest_version ON keys(id, version DESC);

-- 2. Active keys only (90% of queries)
CREATE INDEX idx_keys_active_latest ON keys(id, version DESC) 
WHERE status = 'active';

-- 3. Pagination support
CREATE INDEX idx_keys_created_at ON keys(created_at DESC);

-- 4. Filtered pagination
CREATE INDEX idx_keys_status_created_at ON keys(status, created_at DESC);

-- 5. Keep existing for other operations
CREATE INDEX idx_keys_status ON keys(status);
CREATE INDEX idx_keys_storage_type ON keys(storage_type);
