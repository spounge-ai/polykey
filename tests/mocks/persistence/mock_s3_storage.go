package persistence

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/spounge-ai/polykey/internal/domain"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

type MockS3Storage struct{}

func NewMockS3Storage() domain.KeyRepository {
	return &MockS3Storage{}
}

func (m *MockS3Storage) GetKey(ctx context.Context, id string) (*domain.Key, error) {
	log.Printf("MockS3Storage: GetKey called for id: %s", id)
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

func (m *MockS3Storage) GetKeyByVersion(ctx context.Context, id string, version int32) (*domain.Key, error) {
	log.Printf("MockS3Storage: GetKeyByVersion called for id: %s, version: %d", id, version)
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

func (m *MockS3Storage) CreateKey(ctx context.Context, key *domain.Key, isPremium bool) error {
	log.Printf("MockS3Storage: CreateKey called for key id: %s, isPremium: %t", key.ID, isPremium)
	return nil
}

func (m *MockS3Storage) ListKeys(ctx context.Context) ([]*domain.Key, error) {
	log.Println("MockS3Storage: ListKeys called")
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

func (m *MockS3Storage) UpdateKeyMetadata(ctx context.Context, id string, metadata *pk.KeyMetadata) error {
	log.Printf("MockS3Storage: UpdateKeyMetadata called for id: %s", id)
	return nil
}

func (m *MockS3Storage) RotateKey(ctx context.Context, id string, newEncryptedDEK []byte) (*domain.Key, error) {
	log.Printf("MockS3Storage: RotateKey called for id: %s", id)
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

func (m *MockS3Storage) RevokeKey(ctx context.Context, id string) error {
	log.Printf("MockS3Storage: RevokeKey called for id: %s", id)
	return nil
}

func (m *MockS3Storage) GetKeyVersions(ctx context.Context, id string) ([]*domain.Key, error) {
	log.Printf("MockS3Storage: GetKeyVersions called for id: %s", id)
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
