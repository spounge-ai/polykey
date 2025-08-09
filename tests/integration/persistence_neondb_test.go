package integration_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/infra/persistence"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *persistence.NeonDBStorage {
	cfgPath := os.Getenv("POLYKEY_CONFIG_PATH")
	if cfgPath == "" {
		t.Fatal("POLYKEY_CONFIG_PATH environment variable not set. No configuration file found.")
	}

	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)

	dbpool, err := pgxpool.New(context.Background(), cfg.NeonDB.URL)
	require.NoError(t, err)

	storage, err := persistence.NewNeonDBStorage(dbpool)
	require.NoError(t, err)

	return storage
}

func TestNeonDBStorage(t *testing.T) {
	storage := setupTestDB(t)

	t.Run("Create and Get Key", func(t *testing.T) {
		key := &domain.Key{
			ID:      "test-key-1",
			Version: 1,
			Metadata: &pk.KeyMetadata{
				Description: "Test Key",
			},
			EncryptedDEK: []byte("test-dek"),
			Status:       domain.KeyStatusActive,
		}

		err := storage.CreateKey(context.Background(), key, false)
		require.NoError(t, err)

		retrievedKey, err := storage.GetKey(context.Background(), "test-key-1")
		require.NoError(t, err)
		require.NotNil(t, retrievedKey)
		require.Equal(t, key.ID, retrievedKey.ID)
		require.Equal(t, key.Version, retrievedKey.Version)
	})

	t.Run("Rotate Key", func(t *testing.T) {
		newDEK := []byte("new-test-dek")
		rotatedKey, err := storage.RotateKey(context.Background(), "test-key-1", newDEK)
		require.NoError(t, err)
		require.NotNil(t, rotatedKey)
		require.Equal(t, int32(2), rotatedKey.Version)

		retrievedKey, err := storage.GetKey(context.Background(), "test-key-1")
		require.NoError(t, err)
		require.NotNil(t, retrievedKey)
		require.Equal(t, int32(2), retrievedKey.Version)
		require.Equal(t, newDEK, retrievedKey.EncryptedDEK)
	})
}
