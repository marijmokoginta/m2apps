package config

import (
	"encoding/json"
	"fmt"
	"os"
)

func LoadFromFile(path string) (*InstallConfig, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var cfg InstallConfig
	if err := json.Unmarshal(file, &cfg); err != nil {
		return nil, fmt.Errorf("invalid JSON format: %w", err)
	}

	return &cfg, nil
}
