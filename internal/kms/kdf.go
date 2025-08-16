package kms

import (
	"crypto/sha256"
	"fmt"

	"golang.org/x/crypto/hkdf"
)

// DeriveKey uses HKDF to derive a new key from a master key.
// This is useful for creating specific keys for different purposes without
// exposing the master key.
func DeriveKey(masterKey, salt, info []byte, keyLength int) ([]byte, error) {
	if len(masterKey) == 0 {
		return nil, fmt.Errorf("master key cannot be empty")
	}
	if len(salt) == 0 {
		return nil, fmt.Errorf("salt cannot be empty")
	}

	hkdf := hkdf.New(sha256.New, masterKey, salt, info)

	key := make([]byte, keyLength)
	if _, err := hkdf.Read(key); err != nil {
		return nil, fmt.Errorf("failed to derive key using HKDF: %w", err)
	}

	return key, nil
}
