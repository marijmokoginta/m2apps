package installer

import (
	"fmt"
	"m2apps/internal/logger"
	"m2apps/internal/progress"
	"m2apps/internal/ui"
	"os"
	"path/filepath"
	"strings"

	"m2apps/internal/extractor"
	"m2apps/internal/preset"
)

type InstallContext struct {
	ZipPath         string
	TargetDir       string
	Preset          string
	AppID           string
	ProgressManager *progress.Manager
	ProgressAppID   string

	// DBSetupFunc is an optional hook called after files are extracted and
	// before preset commands run. Use it to inject DB configuration into the
	// .env so that commands like `php artisan migrate` can succeed.
	// workDir is the temporary extraction directory.
	DBSetupFunc func(workDir string) error
}

func Install(ctx InstallContext) error {
	if err := logger.Init(); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Close()

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

	reportProgress(ctx, "install", "extracting files", 70, "Extracting files")
	fmt.Println(ui.Info("[INFO] Extracting files..."))
	if err := extractor.ExtractZip(ctx.ZipPath, tempDir); err != nil {
		return fmt.Errorf("failed to extract zip: %w", err)
	}
	fmt.Println(ui.Success("[OK] Files extracted"))

	// Run DB setup hook (if provided) before preset commands so that
	// commands like `php artisan migrate` have a valid DB config.
	if ctx.DBSetupFunc != nil {
		reportProgress(ctx, "install", "database setup", 74, "Database setup")
		if err := ctx.DBSetupFunc(tempDir); err != nil {
			return fmt.Errorf("database setup failed: %w", err)
		}
	}

	reportProgress(ctx, "install", "running preset", 78, "Running preset")
	fmt.Println(ui.Info("[INFO] Running installation preset..."))
	if err := preset.RunInstallPreset(ctx.Preset, tempDir); err != nil {
		return err
	}
	fmt.Println(ui.Success("[OK] Preset execution completed"))

	reportProgress(ctx, "install", "moving installed files", 86, "Moving installed files")
	fmt.Println(ui.Info("[INFO] Moving installed files..."))
	if err := moveExtractedFiles(tempDir, targetDir); err != nil {
		return fmt.Errorf("failed to move installed files: %w", err)
	}
	fmt.Println(ui.Success("[OK] Files moved to target directory"))

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

		if err := os.RemoveAll(dstPath); err != nil {
			return fmt.Errorf("failed to overwrite target path %s: %w", dstPath, err)
		}

		if err := os.Rename(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to move %s: %w", name, err)
		}
	}

	return nil
}

func reportProgress(ctx InstallContext, phase, step string, percent int, logMessage string) {
	if ctx.ProgressManager == nil || ctx.ProgressAppID == "" {
		return
	}

	ctx.ProgressManager.Update(ctx.ProgressAppID, phase, step, percent)
	if logMessage != "" {
		ctx.ProgressManager.Log(ctx.ProgressAppID, logMessage)
	}
}
