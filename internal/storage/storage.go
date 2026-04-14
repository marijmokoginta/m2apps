package storage

import (
	"encoding/json"
	"fmt"
	"m2apps/internal/system"
	"os"
	"path/filepath"
	"strings"
)

type FileStorage struct {
	baseDir string
}

func New() (Storage, error) {
	baseDir := system.GetBaseDir()
	if err := os.MkdirAll(system.GetAppsDir(), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &FileStorage{baseDir: baseDir}, nil
}

func (s *FileStorage) Save(appID string, data AppConfig) error {
	id := strings.TrimSpace(appID)
	if id == "" {
		return fmt.Errorf("app_id is required")
	}

	appDir := filepath.Join(s.baseDir, "apps", id)
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return fmt.Errorf("failed to create app storage directory: %w", err)
	}

	plain, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to serialize app config: %w", err)
	}

	encrypted, err := encrypt(plain)
	if err != nil {
		return err
	}

	configPath := filepath.Join(appDir, "config.enc")
	if err := os.WriteFile(configPath, encrypted, 0o600); err != nil {
		return fmt.Errorf("failed to write encrypted config: %w", err)
	}

	// Keep state file present for expected storage structure.
	statePath := filepath.Join(appDir, "state.json")
	if err := os.WriteFile(statePath, []byte("{}\n"), 0o600); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

func (s *FileStorage) Load(appID string) (AppConfig, error) {
	id := strings.TrimSpace(appID)
	if id == "" {
		return AppConfig{}, fmt.Errorf("app_id is required")
	}

	configPath := filepath.Join(s.baseDir, "apps", id, "config.enc")
	cipherData, err := os.ReadFile(configPath)
	if err != nil {
		return AppConfig{}, fmt.Errorf("failed to read encrypted config: %w", err)
	}

	plain, err := decrypt(cipherData)
	if err != nil {
		return AppConfig{}, err
	}

	var cfg AppConfig
	if err := json.Unmarshal(plain, &cfg); err != nil {
		return AppConfig{}, fmt.Errorf("failed to deserialize app config: %w", err)
	}

	return cfg, nil
}

func (s *FileStorage) Exists(appID string) (bool, error) {
	id := strings.TrimSpace(appID)
	if id == "" {
		return false, fmt.Errorf("app_id is required")
	}

	appDir := filepath.Join(s.baseDir, "apps", id)
	info, err := os.Stat(appDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check app directory: %w", err)
	}

	return info.IsDir(), nil
}
