package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"os"
)

func decrypt(cipherData []byte) ([]byte, error) {
	plain, _, err := decryptWithMigrationFlag(cipherData)
	return plain, err
}

func decryptWithMigrationFlag(cipherData []byte) ([]byte, bool, error) {
	primaryKey, err := loadPersistentKey(false)
	if err == nil {
		plain, decErr := decryptWithKey(cipherData, primaryKey)
		if decErr == nil {
			return plain, false, nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, false, err
	}

	legacyKeys, legacyErr := deriveLegacyKeys()
	if legacyErr != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, fmt.Errorf("failed to decrypt data: %w", legacyErr)
		}
		return nil, false, fmt.Errorf("failed to decrypt data: %w", err)
	}

	for _, key := range legacyKeys {
		plain, decErr := decryptWithKey(cipherData, key)
		if decErr == nil {
			return plain, true, nil
		}
	}

	return nil, false, fmt.Errorf("failed to decrypt data: cipher: message authentication failed")
}

func decryptWithKey(cipherData, key []byte) ([]byte, error) {
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
	return gcm.Open(nil, nonce, ciphertext, nil)
}
