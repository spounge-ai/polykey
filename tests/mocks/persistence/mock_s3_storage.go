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

func (m *MockS3Storage) GetKey(ctx context.Context, id domain.KeyID) (*domain.Key, error) {
	log.Printf("MockS3Storage: GetKey called for id: %s", id.String())
	if id.String() == "f47ac10b-58cc-4372-a567-0e02b2c3d479" || id.String() == "a47ac10b-58cc-4372-a567-0e02b2c3d479" {
		metadata := &pk.KeyMetadata{KeyId: id.String(), KeyType: pk.KeyType_KEY_TYPE_API_KEY}
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
	return nil, fmt.Errorf("mock: key not found with id: %s", id.String())
}

func (m *MockS3Storage) GetKeyByVersion(ctx context.Context, id domain.KeyID, version int32) (*domain.Key, error) {
	log.Printf("MockS3Storage: GetKeyByVersion called for id: %s, version: %d", id.String(), version)
	if id.String() == "f47ac10b-58cc-4372-a567-0e02b2c3d479" && version == 1 {
		metadata := &pk.KeyMetadata{KeyId: id.String(), KeyType: pk.KeyType_KEY_TYPE_API_KEY}
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
	return nil, fmt.Errorf("mock: key not found with id: %s and version: %d", id.String(), version)
}

func (m *MockS3Storage) CreateKey(ctx context.Context, key *domain.Key, isPremium bool) error {
	log.Printf("MockS3Storage: CreateKey called for key id: %s, isPremium: %t", key.ID.String(), isPremium)
	return nil
}

func (m *MockS3Storage) ListKeys(ctx context.Context) ([]*domain.Key, error) {
	log.Println("MockS3Storage: ListKeys called")
	return []*domain.Key{
		{
			ID:           domain.NewKeyID(),
			Version:      1,
			Metadata:     &pk.KeyMetadata{KeyId: "mock_key_1", KeyType: pk.KeyType_KEY_TYPE_AES_256},
			EncryptedDEK: []byte("mock_encrypted_dek_1"),
			Status:       domain.KeyStatusActive,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			ID:           domain.NewKeyID(),
			Version:      1,
			Metadata:     &pk.KeyMetadata{KeyId: "mock_key_2", KeyType: pk.KeyType_KEY_TYPE_AES_256},
			EncryptedDEK: []byte("mock_encrypted_dek_2"),
			Status:       domain.KeyStatusActive,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}, nil
}

func (m *MockS3Storage) UpdateKeyMetadata(ctx context.Context, id domain.KeyID, metadata *pk.KeyMetadata) error {
	log.Printf("MockS3Storage: UpdateKeyMetadata called for id: %s", id.String())
	return nil
}

func (m *MockS3Storage) RotateKey(ctx context.Context, id domain.KeyID, newEncryptedDEK []byte) (*domain.Key, error) {
	log.Printf("MockS3Storage: RotateKey called for id: %s", id.String())
	return &domain.Key{
		ID:           id,
		Version:      2,
		EncryptedDEK: newEncryptedDEK,
		Metadata:     &pk.KeyMetadata{KeyId: id.String()},
		Status:       domain.KeyStatusRotated,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

func (m *MockS3Storage) RevokeKey(ctx context.Context, id domain.KeyID) error {
	log.Printf("MockS3Storage: RevokeKey called for id: %s", id.String())
	return nil
}

func (m *MockS3Storage) GetKeyVersions(ctx context.Context, id domain.KeyID) ([]*domain.Key, error) {
	log.Printf("MockS3Storage: GetKeyVersions called for id: %s", id.String())
	return []*domain.Key{
		{
			ID:           id,
			Version:      1,
			Metadata:     &pk.KeyMetadata{KeyId: id.String()},
			EncryptedDEK: []byte("mock_dek_v1"),
			Status:       domain.KeyStatusActive,
			CreatedAt:    time.Now().Add(-24 * time.Hour),
			UpdatedAt:    time.Now().Add(-24 * time.Hour),
		},
		{
			ID:           id,
			Version:      2,
			Metadata:     &pk.KeyMetadata{KeyId: id.String()},
			EncryptedDEK: []byte("mock_dek_v2"),
			Status:       domain.KeyStatusRotated,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}, nil
}
