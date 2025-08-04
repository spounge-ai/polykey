package persistence

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/spounge-ai/polykey/internal/domain"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

type MockVaultStorage struct{}

func NewMockVaultStorage() domain.KeyRepository {
	return &MockVaultStorage{}
}

func (m *MockVaultStorage) GetKey(ctx context.Context, id string) (*domain.Key, error) {
	log.Printf("MockVaultStorage: GetKey called for id: %s", id)
	if id == "test_key_123" || id == "test_key_for_metadata" {
		metadata := &pk.KeyMetadata{KeyId: id, KeyType: pk.KeyType_KEY_TYPE_API_KEY}
		return &domain.Key{
			ID:           id,
			Version:      1,
			Metadata:     metadata,
			EncryptedDEK: []byte("mock_encrypted_dek"),
			Status:       domain.KeyStatusActive,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}, nil
	}
	return nil, fmt.Errorf("mock: key not found with id: %s", id)
}

func (m *MockVaultStorage) GetKeyByVersion(ctx context.Context, id string, version int32) (*domain.Key, error) {
	log.Printf("MockVaultStorage: GetKeyByVersion called for id: %s, version: %d", id, version)
	if id == "test_key_123" && version == 1 {
		metadata := &pk.KeyMetadata{KeyId: id, KeyType: pk.KeyType_KEY_TYPE_API_KEY}
		return &domain.Key{
			ID:           id,
			Version:      version,
			Metadata:     metadata,
			EncryptedDEK: []byte("mock_encrypted_dek"),
			Status:       domain.KeyStatusActive,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}, nil
	}
	return nil, fmt.Errorf("mock: key not found with id: %s and version: %d", id, version)
}

func (m *MockVaultStorage) CreateKey(ctx context.Context, key *domain.Key) error {
	log.Printf("MockVaultStorage: CreateKey called for key id: %s", key.ID)
	return nil
}

func (m *MockVaultStorage) ListKeys(ctx context.Context) ([]*domain.Key, error) {
	log.Println("MockVaultStorage: ListKeys called")
	return []*domain.Key{
		{
			ID:           "mock_key_1",
			Version:      1,
			Metadata:     &pk.KeyMetadata{KeyId: "mock_key_1", KeyType: pk.KeyType_KEY_TYPE_AES_256},
			EncryptedDEK: []byte("mock_encrypted_dek_1"),
			Status:       domain.KeyStatusActive,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			ID:           "mock_key_2",
			Version:      1,
			Metadata:     &pk.KeyMetadata{KeyId: "mock_key_2", KeyType: pk.KeyType_KEY_TYPE_AES_256},
			EncryptedDEK: []byte("mock_encrypted_dek_2"),
			Status:       domain.KeyStatusActive,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}, nil
}

func (m *MockVaultStorage) UpdateKeyMetadata(ctx context.Context, id string, metadata *pk.KeyMetadata) error {
	log.Printf("MockVaultStorage: UpdateKeyMetadata called for id: %s", id)
	return nil
}

func (m *MockVaultStorage) RotateKey(ctx context.Context, id string, newEncryptedDEK []byte) (*domain.Key, error) {
	log.Printf("MockVaultStorage: RotateKey called for id: %s", id)
	return &domain.Key{
		ID:           id,
		Version:      2,
		EncryptedDEK: newEncryptedDEK,
		Metadata:     &pk.KeyMetadata{KeyId: id},
		Status:       domain.KeyStatusRotated,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

func (m *MockVaultStorage) RevokeKey(ctx context.Context, id string) error {
	log.Printf("MockVaultStorage: RevokeKey called for id: %s", id)
	return nil
}

func (m *MockVaultStorage) GetKeyVersions(ctx context.Context, id string) ([]*domain.Key, error) {
	log.Printf("MockVaultStorage: GetKeyVersions called for id: %s", id)
	return []*domain.Key{
		{
			ID:           id,
			Version:      1,
			Metadata:     &pk.KeyMetadata{KeyId: id},
			EncryptedDEK: []byte("mock_dek_v1"),
			Status:       domain.KeyStatusActive,
			CreatedAt:    time.Now().Add(-24 * time.Hour),
			UpdatedAt:    time.Now().Add(-24 * time.Hour),
		},
		{
			ID:           id,
			Version:      2,
			Metadata:     &pk.KeyMetadata{KeyId: id},
			EncryptedDEK: []byte("mock_dek_v2"),
			Status:       domain.KeyStatusRotated,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}, nil
}
