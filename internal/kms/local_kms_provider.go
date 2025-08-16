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

// EncryptDEK encrypts the given plaintext DEK using a derived key.
func (p *LocalKMSProvider) EncryptDEK(ctx context.Context, plaintextDEK []byte, key *domain.Key) ([]byte, error) {
	return execution.WithTimeout(ctx, localKmsTimeout, func(ctx context.Context) ([]byte, error) {
		info := []byte(key.ID.String())
		salt := []byte("polykey-salt:" + key.ID.String())
		derivedKey, err := DeriveKey(p.masterKey, salt, info, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to derive key: %w", err)
		}

		return p.encryptWithKey(derivedKey, plaintextDEK)
	})
}

// DecryptDEK decrypts the DEK using a derived key, with a fallback to the master key for backward compatibility.
func (p *LocalKMSProvider) DecryptDEK(ctx context.Context, key *domain.Key) ([]byte, error) {
	return execution.WithTimeout(ctx, localKmsTimeout, func(ctx context.Context) ([]byte, error) {
		// First, try decrypting with the derived key (the new method).
		info := []byte(key.ID.String())
		salt := []byte("polykey-salt:" + key.ID.String())
		derivedKey, err := DeriveKey(p.masterKey, salt, info, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to derive key for decryption: %w", err)
		}

		plaintextDEK, err := p.decryptWithKey(derivedKey, key.EncryptedDEK)
		if err == nil {
			return plaintextDEK, nil // Success with the new method
		}

		// If decryption with the derived key fails, fall back to the old method (master key).
		// This provides backward compatibility for keys encrypted before the KDF was introduced.
		return p.decryptWithKey(p.masterKey, key.EncryptedDEK)
	})
}

func (p *LocalKMSProvider) encryptWithKey(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
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

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func (p *LocalKMSProvider) decryptWithKey(key, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, fmt.Errorf("no encrypted DEK found in key")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("encrypted DEK too short")
	}

	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, actualCiphertext, nil)
}

func (p *LocalKMSProvider) HealthCheck(ctx context.Context) error {
	return nil
}
