package kms

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// LocalKMSService is a production-ready implementation of the KMSService interface that uses a local key for encryption and decryption.
// This is intended for use in free tiers or local development where KMS is not available.

type LocalKMSService struct {
	masterKey []byte
}

func NewLocalKMSService(masterKey string) (*LocalKMSService, error) {
	key, err := base64.StdEncoding.DecodeString(masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode master key: %w", err)
	}
	return &LocalKMSService{masterKey: key}, nil
}

func (s *LocalKMSService) EncryptDEK(ctx context.Context, plaintextDEK []byte, isPremium bool) ([]byte, error) {
	if isPremium {
		return nil, fmt.Errorf("cannot use local kms for premium keys")
	}
	block, err := aes.NewCipher(s.masterKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintextDEK, nil)
	return ciphertext, nil
}

func (s *LocalKMSService) DecryptDEK(ctx context.Context, encryptedDEK []byte, isPremium bool) ([]byte, error) {
	if isPremium {
		return nil, fmt.Errorf("cannot use local kms for premium keys")
	}
	block, err := aes.NewCipher(s.masterKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(encryptedDEK) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := encryptedDEK[:nonceSize], encryptedDEK[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
