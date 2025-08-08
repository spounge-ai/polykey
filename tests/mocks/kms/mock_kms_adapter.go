package kms

import (
	"context"

	"github.com/spounge-ai/polykey/internal/domain"
)

// MockKMSAdapter is a placeholder for a mock KMS service.
type MockKMSAdapter struct{}

// NewMockKMSAdapter creates a new mock KMS adapter.
func NewMockKMSAdapter() domain.KMSService {
	return &MockKMSAdapter{}
}

// EncryptDEK is a mock implementation of the EncryptDEK method.
func (m *MockKMSAdapter) EncryptDEK(ctx context.Context, key *domain.Key) ([]byte, error) {
	return []byte("mock_encrypted_dek"), nil
}

// DecryptDEK is a mock implementation of the DecryptDEK method.
func (m *MockKMSAdapter) DecryptDEK(ctx context.Context, key *domain.Key) ([]byte, error) {
	return []byte("mock_plaintext_dek"), nil
}
