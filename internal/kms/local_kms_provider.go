package kms

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	

	"github.com/spounge-ai/polykey/internal/domain"
)

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

func (p *LocalKMSProvider) EncryptDEK(ctx context.Context, key *domain.Key) ([]byte, error) {
	if key.IsPremium() {
		return nil, fmt.Errorf("cannot use local kms for premium keys")
	}
	block, err := aes.NewCipher(p.masterKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, key.EncryptedDEK, nil)
	return ciphertext, nil
}

func (p *LocalKMSProvider) DecryptDEK(ctx context.Context, key *domain.Key) ([]byte, error) {
	if key.IsPremium() {
		return nil, fmt.Errorf("cannot use local kms for premium keys")
	}
	block, err := aes.NewCipher(p.masterKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(key.EncryptedDEK) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := key.EncryptedDEK[:nonceSize], key.EncryptedDEK[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}