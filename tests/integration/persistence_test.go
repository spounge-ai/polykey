
package integration_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/internal/infra/persistence"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"github.com/stretchr/testify/require"
)

func setupPersistence(t *testing.T) (*persistence.PSQLAdapter, func()) {
	t.Helper()
	adapter, err := persistence.NewPSQLAdapter(dbpool, slog.Default())
	require.NoError(t, err)

	cleanup := func() {
		truncate(t)
	}

	return adapter, cleanup
}

func TestPersistence_CreateAndGetKey(t *testing.T) {
	adapter, cleanup := setupPersistence(t)
	defer cleanup()

	ctx := context.Background()
	keyID := domain.NewKeyID()
	key := &domain.Key{
		ID:      keyID,
		Version: 1,
		Metadata: &pk.KeyMetadata{
			Description: "test key",
			KeyType:     pk.KeyType_KEY_TYPE_AES_256,
		},
		EncryptedDEK: []byte("encrypted-dek"),
		Status:       domain.KeyStatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err := adapter.CreateKey(ctx, key)
	require.NoError(t, err)

	retrievedKey, err := adapter.GetKey(ctx, keyID)
	require.NoError(t, err)
	require.NotNil(t, retrievedKey)
	require.Equal(t, key.ID, retrievedKey.ID)
	require.Equal(t, key.Version, retrievedKey.Version)
	require.Equal(t, key.Metadata.Description, retrievedKey.Metadata.Description)
	require.Equal(t, key.EncryptedDEK, retrievedKey.EncryptedDEK)
	require.Equal(t, key.Status, retrievedKey.Status)
}

func TestPersistence_RotateKey(t *testing.T) {
	adapter, cleanup := setupPersistence(t)
	defer cleanup()

	ctx := context.Background()
	keyID := domain.NewKeyID()
	key := &domain.Key{
		ID:      keyID,
		Version: 1,
		Metadata: &pk.KeyMetadata{
			Description: "key to rotate",
			KeyType:     pk.KeyType_KEY_TYPE_AES_256,
			Version:     1,
		},
		EncryptedDEK: []byte("initial-dek"),
		Status:       domain.KeyStatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err := adapter.CreateKey(ctx, key)
	require.NoError(t, err)

	newDEK := []byte("rotated-dek")
	rotatedKey, err := adapter.RotateKey(ctx, keyID, newDEK)
	require.NoError(t, err)
	require.NotNil(t, rotatedKey)
	require.Equal(t, int32(2), rotatedKey.Version)
	require.Equal(t, newDEK, rotatedKey.EncryptedDEK)

	latestKey, err := adapter.GetKey(ctx, keyID)
	require.NoError(t, err)
	require.Equal(t, int32(2), latestKey.Version)

	v1Key, err := adapter.GetKeyByVersion(ctx, keyID, 1)
	require.NoError(t, err)
	require.Equal(t, domain.KeyStatusRotated, v1Key.Status)
}

func TestPersistence_UpdateKeyMetadata(t *testing.T) {
	adapter, cleanup := setupPersistence(t)
	defer cleanup()

	ctx := context.Background()
	keyID := domain.NewKeyID()
	key := &domain.Key{
		ID:      keyID,
		Version: 1,
		Metadata: &pk.KeyMetadata{
			Description: "initial description",
			KeyType:     pk.KeyType_KEY_TYPE_AES_256,
		},
		EncryptedDEK: []byte("encrypted-dek"),
		Status:       domain.KeyStatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err := adapter.CreateKey(ctx, key)
	require.NoError(t, err)

	updatedMetadata := &pk.KeyMetadata{
		Description: "updated description",
		Tags:        map[string]string{"a": "b"},
	}

	err = adapter.UpdateKeyMetadata(ctx, keyID, updatedMetadata)
	require.NoError(t, err)

	retrievedKey, err := adapter.GetKey(ctx, keyID)
	require.NoError(t, err)
	require.Equal(t, "updated description", retrievedKey.Metadata.Description)
	require.Equal(t, "b", retrievedKey.Metadata.Tags["a"])
}

func TestPersistence_RevokeKey(t *testing.T) {
	adapter, cleanup := setupPersistence(t)
	defer cleanup()

	ctx := context.Background()
	keyID := domain.NewKeyID()
	key := &domain.Key{
		ID:      keyID,
		Version: 1,
		Metadata: &pk.KeyMetadata{
			Description: "key to revoke",
			KeyType:     pk.KeyType_KEY_TYPE_AES_256,
		},
		EncryptedDEK: []byte("encrypted-dek"),
		Status:       domain.KeyStatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err := adapter.CreateKey(ctx, key)
	require.NoError(t, err)

	err = adapter.RevokeKey(ctx, keyID)
	require.NoError(t, err)

	retrievedKey, err := adapter.GetKey(ctx, keyID)
	require.NoError(t, err)
	require.Equal(t, domain.KeyStatusRevoked, retrievedKey.Status)
	require.NotNil(t, retrievedKey.RevokedAt)
}
