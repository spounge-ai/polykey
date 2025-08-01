package domain

import "context"

// KMSService defines the interface for interacting with a Key Management Service.
type KMSService interface {
	EncryptDEK(ctx context.Context, plaintextDEK []byte, masterKeyID string) ([]byte, error)
	DecryptDEK(ctx context.Context, encryptedDEK []byte, masterKeyID string) ([]byte, error)
}
