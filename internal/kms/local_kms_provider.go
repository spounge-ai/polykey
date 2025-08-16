package kms

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/pkg/execution"
)

const localKmsTimeout = 1 * time.Second

type LocalKMSProvider struct {
	masterKey []byte
}

func NewLocalKMSProvider(masterKey string) (*LocalKMSProvider, error) {
	key, err := base64.StdEncoding.DecodeString(masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode master key: %w", err)
	}
	return &LocalKMSProvider{masterKey: key}, nil
}

// EncryptDEK encrypts the given plaintext DEK and returns the encrypted DEK.
func (p *LocalKMSProvider) EncryptDEK(ctx context.Context, plaintextDEK []byte, key *domain.Key) ([]byte, error) {
	return execution.WithTimeout(ctx, localKmsTimeout, func(ctx context.Context) ([]byte, error) {
		// Encrypt the DEK with our master key
		block, err := aes.NewCipher(p.masterKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create cipher: %w", err)
		}

		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return nil, fmt.Errorf("failed to create GCM: %w", err)
		}

		nonce := make([]byte, gcm.NonceSize())
		if _, err := rand.Read(nonce); err != nil {
			return nil, fmt.Errorf("failed to generate nonce: %w", err)
		}

		encryptedDEK := gcm.Seal(nonce, nonce, plaintextDEK, nil)
		return encryptedDEK, nil
	})
}

// DecryptDEK takes the encrypted DEK from the key and returns the plaintext DEK
func (p *LocalKMSProvider) DecryptDEK(ctx context.Context, key *domain.Key) ([]byte, error) {
	return execution.WithTimeout(ctx, localKmsTimeout, func(ctx context.Context) ([]byte, error) {
		if len(key.EncryptedDEK) == 0 {
			return nil, fmt.Errorf("no encrypted DEK found in key")
		}

		block, err := aes.NewCipher(p.masterKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create cipher: %w", err)
		}

		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return nil, fmt.Errorf("failed to create GCM: %w", err)
		}

		nonceSize := gcm.NonceSize()
		if len(key.EncryptedDEK) < nonceSize {
			return nil, fmt.Errorf("encrypted DEK too short")
		}

		nonce, ciphertext := key.EncryptedDEK[:nonceSize], key.EncryptedDEK[nonceSize:]
		plaintextDEK, err := gcm.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt DEK: %w", err)
		}

		return plaintextDEK, nil
	})
}

func (p *LocalKMSProvider) HealthCheck(ctx context.Context) error {
	return nil
}