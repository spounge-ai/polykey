CREATE TABLE IF NOT EXISTS keys (
    id VARCHAR(255) NOT NULL,
    version INT NOT NULL,
    metadata JSONB,
    encrypted_dek BYTEA,
    status VARCHAR(50),
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    is_premium BOOLEAN DEFAULT FALSE,
    PRIMARY KEY (id, version)
);