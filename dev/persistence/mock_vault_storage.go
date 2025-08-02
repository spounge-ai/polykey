package persistence

import (
	"context"
	"fmt"
	"log"

	"github.com/spounge-ai/polykey/internal/domain"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// MockVaultStorage implements the domain.KeyRepository interface for testing.
type MockVaultStorage struct{}

// NewMockVaultStorage creates a new MockVaultStorage instance.
func NewMockVaultStorage() domain.KeyRepository {
	return &MockVaultStorage{}
}

// GetKey retrieves a key from mock storage.
func (m *MockVaultStorage) GetKey(ctx context.Context, id string) (*domain.Key, error) {
	log.Printf("MockVaultStorage: GetKey called for id: %s", id)
	if id == "test_key_123" || id == "test_key_for_metadata" {
		metadata := &pk.KeyMetadata{KeyId: id, KeyType: pk.KeyType_KEY_TYPE_API_KEY}
		return &domain.Key{
			ID: id,
			Metadata: metadata,
			EncryptedDEK: []byte("mock_encrypted_dek"),
		}, nil
	}
	return nil, fmt.Errorf("mock: key not found with id: %s", id)
}

// CreateKey stores a key in mock storage.
func (m *MockVaultStorage) CreateKey(ctx context.Context, key *domain.Key) error {
	log.Printf("MockVaultStorage: CreateKey called for key id: %s", key.ID)
	return nil
}

// ListKeys lists keys in mock storage.
func (m *MockVaultStorage) ListKeys(ctx context.Context) ([]*domain.Key, error) {
	log.Println("MockVaultStorage: ListKeys called")
	return []*domain.Key{
		{
			ID: "mock_key_1",
			Metadata: &pk.KeyMetadata{KeyId: "mock_key_1", KeyType: pk.KeyType_KEY_TYPE_AES_256},
			EncryptedDEK: []byte("mock_encrypted_dek_1"),
		},
		{
			ID: "mock_key_2",
			Metadata: &pk.KeyMetadata{KeyId: "mock_key_2", KeyType: pk.KeyType_KEY_TYPE_AES_256},
			EncryptedDEK: []byte("mock_encrypted_dek_2"),
		},
	}, nil
}


