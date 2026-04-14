package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/user"
	"strings"
)

const staticSalt = "m2apps_static_salt"

func encrypt(plain []byte) ([]byte, error) {
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

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plain, nil)
	return append(nonce, ciphertext...), nil
}

func deriveKey() ([]byte, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to read hostname: %w", err)
	}

	username, err := currentUsername()
	if err != nil {
		return nil, err
	}

	material := hostname + username + staticSalt
	hash := sha256.Sum256([]byte(material))
	return hash[:], nil
}

func currentUsername() (string, error) {
	if name := strings.TrimSpace(os.Getenv("USER")); name != "" {
		return name, nil
	}
	if name := strings.TrimSpace(os.Getenv("USERNAME")); name != "" {
		return name, nil
	}

	u, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to resolve current user: %w", err)
	}
	if strings.TrimSpace(u.Username) == "" {
		return "", fmt.Errorf("failed to resolve current username")
	}
	return u.Username, nil
}
