package domain

import "context"

type KMSService interface {
	EncryptDEK(ctx context.Context, plaintextDEK []byte, isPremium bool) ([]byte, error)
	DecryptDEK(ctx context.Context, encryptedDEK []byte, isPremium bool) ([]byte, error)
}
