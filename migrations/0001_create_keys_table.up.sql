CREATE TABLE IF NOT EXISTS keys (
    id VARCHAR(255) NOT NULL,
    version INT NOT NULL,
    metadata JSONB,
    encrypted_dek BYTEA,
    status VARCHAR(50),
    is_premium BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    PRIMARY KEY (id, version)
);