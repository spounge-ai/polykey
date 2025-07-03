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
	// regex or some pattern/model based valid for multiple models
	// call providers once made 

}
 
func generateNonce(size int)([]byte, error) {
	nonce := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	return nonce, nil
}


func encryptKey(key []byte, plaintext []byte) ([]byte, error) {
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




func decryptKey(key []byte, ciphertext []byte) ([]byte, error) {

	block, err := aes.NewCipher(key)
	if err != nil{
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM cipher: %w", err)
	}

	nonceSize := aesgcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short: %d bytes, expected at least %d bytes", len(ciphertext), nonceSize)
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	return plaintext, nil
}



/* 
	Multi-insertion: 
*/


func Encrypt(key []byte, plaintext []byte) ([]byte, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}
	return encryptKey(key, plaintext)
}

func Decrypt(key []byte, ciphertext []byte) ([]byte, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}
	return decryptKey(key, ciphertext)
}





func BatchEncrypt(key []byte, plaintexts [][]byte) ([][]byte, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}

	encrypted := make([][]byte, 0, len(plaintexts))
	for _, plaintext := range plaintexts {
		ciphertext, err := encryptKey(key, plaintext)
		if err != nil{
			return nil, fmt.Errorf("failed to encrypt plaintext: %w", err)
		}
		encrypted = append(encrypted, ciphertext)
	}
	return encrypted, nil
}

func BatchDecrypt(key []byte, ciphertext [][]byte) ([][]byte, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}

	decrypted := make([][]byte, 0, len(ciphertext))

	for _, cipher := range ciphertext {
		plaintext, err := decryptKey(key, cipher)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt ciphertext: %w", err)
		}
		decrypted = append(decrypted, plaintext)
	}
	return decrypted, nil
}