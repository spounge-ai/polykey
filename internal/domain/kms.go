package domain

import "context"

type KMSService interface {
	EncryptDEK(ctx context.Context, plaintextDEK []byte, masterKeyID string) ([]byte, error)
	DecryptDEK(ctx context.Context, encryptedDEK []byte, masterKeyID string) ([]byte, error)
}
