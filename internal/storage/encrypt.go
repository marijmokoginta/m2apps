package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"m2apps/internal/system"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

const staticSalt = "m2apps_static_salt"
const storageKeyFilename = "storage.key"

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
	return loadPersistentKey(true)
}

func loadPersistentKey(createIfMissing bool) ([]byte, error) {
	keyPath := storageKeyPath()
	data, err := os.ReadFile(keyPath)
	if err == nil {
		return parseStoredKey(data)
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read storage key: %w", err)
	}
	if !createIfMissing {
		return nil, os.ErrNotExist
	}

	if err := os.MkdirAll(system.GetBaseDir(), 0o700); err != nil {
		return nil, fmt.Errorf("failed to prepare storage key directory: %w", err)
	}

	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate storage key: %w", err)
	}

	encoded := hex.EncodeToString(key) + "\n"
	if err := os.WriteFile(keyPath, []byte(encoded), 0o600); err != nil {
		return nil, fmt.Errorf("failed to write storage key: %w", err)
	}
	return key, nil
}

func parseStoredKey(data []byte) ([]byte, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, fmt.Errorf("invalid storage key: empty")
	}
	raw, err := hex.DecodeString(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid storage key format: %w", err)
	}
	if len(raw) != 32 {
		return nil, fmt.Errorf("invalid storage key length")
	}
	return raw, nil
}

func storageKeyPath() string {
	return filepath.Join(system.GetBaseDir(), storageKeyFilename)
}

func deriveLegacyKeys() ([][]byte, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to read hostname: %w", err)
	}

	seenUsers := map[string]struct{}{}
	addUser := func(name string) {
		n := strings.TrimSpace(name)
		if n == "" {
			return
		}
		seenUsers[n] = struct{}{}

		if strings.Contains(n, "\\") {
			parts := strings.Split(n, "\\")
			seenUsers[strings.TrimSpace(parts[len(parts)-1])] = struct{}{}
		}
		if strings.Contains(n, "/") {
			parts := strings.Split(n, "/")
			seenUsers[strings.TrimSpace(parts[len(parts)-1])] = struct{}{}
		}

		seenUsers[strings.ToLower(n)] = struct{}{}
	}

	addUser(os.Getenv("USER"))
	addUser(os.Getenv("USERNAME"))
	if u, err := user.Current(); err == nil {
		addUser(u.Username)
		addUser(u.Name)
	}

	if len(seenUsers) == 0 {
		return nil, fmt.Errorf("failed to resolve any legacy username candidates")
	}

	keys := make([][]byte, 0, len(seenUsers))
	seenKeys := map[string]struct{}{}
	for username := range seenUsers {
		material := hostname + username + staticSalt
		hash := sha256.Sum256([]byte(material))
		keyHex := hex.EncodeToString(hash[:])
		if _, ok := seenKeys[keyHex]; ok {
			continue
		}
		seenKeys[keyHex] = struct{}{}
		key := make([]byte, 32)
		copy(key, hash[:])
		keys = append(keys, key)
	}

	return keys, nil
}
