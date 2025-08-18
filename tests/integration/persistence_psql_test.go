package integration_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/infra/persistence"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*persistence.PSQLAdapter, func()) {
	cfgPath := os.Getenv("POLYKEY_CONFIG_PATH")
	if cfgPath == "" {
		t.Fatal("POLYKEY_CONFIG_PATH environment variable not set. No configuration file found.")
	}

	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)

	dbpool, err := pgxpool.New(context.Background(), cfg.BootstrapSecrets.NeonDBURLDevelopment)
	require.NoError(t, err)

	storage, err := persistence.NewPSQLAdapter(dbpool, slog.Default())
	require.NoError(t, err)

	cleanup := func() {
		_, err := dbpool.Exec(context.Background(), "DELETE FROM keys")
		require.NoError(t, err)
	}

	return storage, cleanup
}

func TestNeonDBStorage(t *testing.T) {
	storage, cleanup := setupTestDB(t)

	t.Run("Create and Get Key", func(t *testing.T) {
		cleanup()
		keyID, err := domain.KeyIDFromString("f47ac10b-58cc-4372-a567-0e02b2c3d479")
		require.NoError(t, err)

		key := &domain.Key{
			ID:      keyID,
			Version: 1,
			Metadata: &pk.KeyMetadata{
				Description: "Test Key",
			},
			EncryptedDEK: []byte("test-dek"),
			Status:       domain.KeyStatusActive,
		}

		_, err = storage.CreateKey(context.Background(), key)
		require.NoError(t, err)

		retrievedKey, err := storage.GetKey(context.Background(), keyID)
		require.NoError(t, err)
		require.NotNil(t, retrievedKey)
		require.Equal(t, key.ID, retrievedKey.ID)
		require.Equal(t, key.Version, retrievedKey.Version)
	})

	t.Run("Rotate Key", func(t *testing.T) {
		cleanup()
		keyID, err := domain.KeyIDFromString("b47ac10b-58cc-4372-a567-0e02b2c3d479")
		require.NoError(t, err)

		key := &domain.Key{
			ID:      keyID,
			Version: 1,
			Metadata: &pk.KeyMetadata{
				Description: "Test Key to be rotated",
			},
			EncryptedDEK: []byte("initial-dek"),
			Status:       domain.KeyStatusActive,
		}
		_, err = storage.CreateKey(context.Background(), key)
		require.NoError(t, err)

		newDEK := []byte("new-test-dek")
		rotatedKey, err := storage.RotateKey(context.Background(), keyID, newDEK)
		require.NoError(t, err)
		require.NotNil(t, rotatedKey)
		require.Equal(t, int32(2), rotatedKey.Version)

		retrievedKey, err := storage.GetKey(context.Background(), keyID)
		require.NoError(t, err)
		require.NotNil(t, retrievedKey)
		require.Equal(t, int32(2), retrievedKey.Version)
		require.Equal(t, newDEK, retrievedKey.EncryptedDEK)
	})
}
