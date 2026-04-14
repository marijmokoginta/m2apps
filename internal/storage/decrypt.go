package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
)

func decrypt(cipherData []byte) ([]byte, error) {
	key, err := deriveKey()
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GCM mode: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(cipherData) < nonceSize {
		return nil, fmt.Errorf("invalid encrypted data")
	}

	nonce := cipherData[:nonceSize]
	ciphertext := cipherData[nonceSize:]

	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return plain, nil
}
