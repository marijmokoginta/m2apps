package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"m2apps/internal/extractor"
	"m2apps/internal/preset"
)

type InstallContext struct {
	ZipPath   string
	TargetDir string
	Preset    string
	AppID     string
}

func Install(ctx InstallContext) error {
	appID := strings.TrimSpace(ctx.AppID)
	if appID == "" {
		return fmt.Errorf("app_id is required")
	}

	targetDir := strings.TrimSpace(ctx.TargetDir)
	if targetDir == "" {
		return fmt.Errorf("target directory is required")
	}

	tempDir := filepath.Join(targetDir, ".m2apps_tmp", appID)

	if err := os.RemoveAll(tempDir); err != nil {
		return fmt.Errorf("failed to clean temp directory: %w", err)
	}
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := extractor.ExtractZip(ctx.ZipPath, tempDir); err != nil {
		return fmt.Errorf("failed to extract zip: %w", err)
	}

	steps, err := preset.GetPreset(ctx.Preset)
	if err != nil {
		return err
	}

	if err := preset.RunSteps(steps, tempDir); err != nil {
		return err
	}

	if err := moveExtractedFiles(tempDir, targetDir); err != nil {
		return fmt.Errorf("failed to move installed files: %w", err)
	}

	return nil
}

func moveExtractedFiles(fromDir string, toDir string) error {
	entries, err := os.ReadDir(fromDir)
	if err != nil {
		return fmt.Errorf("failed to read temp directory: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		srcPath := filepath.Join(fromDir, name)
		dstPath := filepath.Join(toDir, name)

		if _, err := os.Stat(dstPath); err == nil {
			return fmt.Errorf("target path already exists: %s", dstPath)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("failed to check target path %s: %w", dstPath, err)
		}

		if err := os.Rename(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to move %s: %w", name, err)
		}
	}

	return nil
}
