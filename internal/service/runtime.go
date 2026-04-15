package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func resolveExecutablePath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to resolve executable path: %w", err)
	}
	execPath = filepath.Clean(strings.TrimSpace(execPath))
	if execPath == "" {
		return "", fmt.Errorf("failed to resolve executable path")
	}
	return execPath, nil
}
