package constants

// Prepared statement names
const (
	StmtGetLatestKey    = "get_latest_key"
	StmtGetKeyByVersion = "get_key_by_version"
	StmtCreateKey       = "create_key"
	StmtUpdateMetadata  = "update_metadata"
	StmtRevokeKey       = "revoke_key"
	StmtCheckExists     = "check_exists"
	StmtGetVersions     = "get_versions"
	StmtListKeys        = "list_keys"
)

var Queries = map[string]string{
	StmtGetLatestKey: `
		SELECT version, metadata, encrypted_dek, status, storage_type, created_at, updated_at, revoked_at 
		FROM keys 
		WHERE id = $1::uuid 
		ORDER BY version DESC 
		LIMIT 1`,

	StmtGetKeyByVersion: `
		SELECT metadata, encrypted_dek, status, storage_type, created_at, updated_at, revoked_at 
		FROM keys 
		WHERE id = $1::uuid AND version = $2`,

	StmtCreateKey: `
		INSERT INTO keys (id, version, metadata, encrypted_dek, status, storage_type, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING version, created_at, updated_at`,

	StmtUpdateMetadata: `
		UPDATE keys 
		SET metadata = $1, updated_at = $2 
		WHERE id = $3::uuid AND version = (
			SELECT MAX(version) FROM keys WHERE id = $3::uuid
		)`,

	StmtRevokeKey: `
		UPDATE keys 
		SET status = $1, revoked_at = $2 
		WHERE id = $3::uuid`,

	StmtCheckExists: `
		SELECT EXISTS(SELECT 1 FROM keys WHERE id = $1::uuid LIMIT 1)`,

	StmtGetVersions: `
		SELECT version, metadata, encrypted_dek, status, storage_type, created_at, updated_at, revoked_at 
		FROM keys 
		WHERE id = $1::uuid 
		ORDER BY version DESC`,

	StmtListKeys: `
		WITH latest_keys AS (
			SELECT DISTINCT ON (id) id, version, metadata, encrypted_dek, status, storage_type, 
				   created_at, updated_at, revoked_at
			FROM keys 
			ORDER BY id, version DESC
		)
		SELECT id, version, metadata, encrypted_dek, status, storage_type, 
			   created_at, updated_at, revoked_at 
		FROM latest_keys
		ORDER BY created_at DESC`,
}
