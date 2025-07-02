package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

// AES-256-GCM based
// reference: https://gist.github.com/kkirsche/e28da6754c39d5e7ea10


func validateKey(key[]byte) (error){
	if len(key) != 32{
		return fmt.Errorf("key length must be 32 bytes, got %d bytes", len(key))
	}
	return nil
}
 
func generateNonce(size int)([]byte, error) {
	nonce := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	return nonce, nil
}


func Encrypt(key []byte, plaintext []byte) ([]byte, error) {
	validateKey(key)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM cipher: %w", err)
	}

	nonce, err := generateNonce(aesgcm.NonceSize())
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}


	ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)
	output := append(nonce, ciphertext...)

	return output, nil

}