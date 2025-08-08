package persistence

import (
	"context"

	"github.com/spounge-ai/polykey/internal/domain"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

type NeonDBStorage struct {
	// DB connection
}

func NewNeonDBStorage() (*NeonDBStorage, error) {
	// Initialize DB connection
	return &NeonDBStorage{}, nil
}

func (s *NeonDBStorage) GetKey(ctx context.Context, id string) (*domain.Key, error) {
	return nil, nil
}

func (s *NeonDBStorage) GetKeyByVersion(ctx context.Context, id string, version int32) (*domain.Key, error) {
	return nil, nil
}

func (s *NeonDBStorage) CreateKey(ctx context.Context, key *domain.Key) error {
	return nil
}

func (s *NeonDBStorage) ListKeys(ctx context.Context) ([]*domain.Key, error) {
	return nil, nil
}

func (s *NeonDBStorage) UpdateKeyMetadata(ctx context.Context, id string, metadata *pk.KeyMetadata) error {
	return nil
}

func (s *NeonDBStorage) RotateKey(ctx context.Context, id string, newEncryptedDEK []byte) (*domain.Key, error) {
	return nil, nil
}

func (s *NeonDBStorage) RevokeKey(ctx context.Context, id string) error {
	return nil
}

func (s *NeonDBStorage) GetKeyVersions(ctx context.Context, id string) ([]*domain.Key, error) {
	return nil, nil
}
