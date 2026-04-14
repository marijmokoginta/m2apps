package updater

import (
	"fmt"
	"m2apps/internal/downloader"
	"m2apps/internal/github"
	"m2apps/internal/installer"
	"m2apps/internal/storage"
	"m2apps/internal/ui"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func Update(appID string) error {
	id := strings.TrimSpace(appID)
	if id == "" {
		return fmt.Errorf("app_id is required")
	}

	store, err := storage.New()
	if err != nil {
		return err
	}

	config, err := store.Load(id)
	if err != nil {
		return fmt.Errorf("failed to load app metadata: %w", err)
	}

	channel := github.NormalizeChannel(config.Channel)
	fmt.Println(ui.Info(fmt.Sprintf("[INFO] Checking update (channel: %s)...", channel)))

	owner, repo, err := github.ParseRepo(config.Repo)
	if err != nil {
		return err
	}

	client := github.NewClient(config.Token)
	target, err := github.SelectLatestReleaseByChannel(client, owner, repo, channel)
	if err != nil {
		return err
	}
	fmt.Println(ui.Info(fmt.Sprintf("[INFO] Current: %s", config.Version)))
	fmt.Println(ui.Info(fmt.Sprintf("[INFO] Latest:  %s", target.TagName)))

	newer, err := IsNewer(target.TagName, config.Version)
	if err != nil {
		return fmt.Errorf("failed to compare versions: %w", err)
	}
	if !newer {
		fmt.Println(ui.Success("[OK] Already up to date"))
		return nil
	}

	asset, err := github.FindAsset(target, config.Asset)
	if err != nil {
		return fmt.Errorf("Asset not found in selected release")
	}

	stageRoot := filepath.Join(filepath.Dir(config.InstallPath), ".m2apps_update_stage", config.AppID)
	if err := os.RemoveAll(stageRoot); err != nil {
		return fmt.Errorf("failed to clean update stage directory: %w", err)
	}
	if err := os.MkdirAll(stageRoot, 0o755); err != nil {
		return fmt.Errorf("failed to create update stage directory: %w", err)
	}
	defer os.RemoveAll(stageRoot)

	downloadPath := filepath.Join(stageRoot, "downloads", asset.Name)
	if err := os.MkdirAll(filepath.Dir(downloadPath), 0o755); err != nil {
		return fmt.Errorf("failed to create download directory: %w", err)
	}

	fmt.Println(ui.Info("[INFO] Downloading update..."))
	dl := downloader.New(config.Token)
	if err := dl.Download(asset.URL, downloadPath, printUpdateProgress); err != nil {
		return err
	}
	fmt.Println()
	fmt.Println(ui.Success("[OK] Update package downloaded"))

	candidateDir := filepath.Join(stageRoot, "candidate")
	installCtx := installer.InstallContext{
		ZipPath:   downloadPath,
		TargetDir: candidateDir,
		Preset:    config.Preset,
		AppID:     config.AppID,
	}

	if err := installer.Install(installCtx); err != nil {
		return fmt.Errorf("failed to install update package: %w", err)
	}

	if err := replaceInstall(candidateDir, config.InstallPath); err != nil {
		return err
	}

	config.Version = target.TagName
	config.Channel = channel
	if err := store.Save(config.AppID, config); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	fmt.Println(ui.Success("[OK] Update completed"))
	return nil
}

func replaceInstall(candidatePath string, installPath string) error {
	current := filepath.Clean(strings.TrimSpace(installPath))
	if current == "" {
		return fmt.Errorf("install path is empty")
	}

	if _, err := os.Stat(current); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("install path does not exist: %s", current)
		}
		return fmt.Errorf("failed to access install path %s: %w", current, err)
	}

	backup := current + ".m2apps_backup"
	if err := os.RemoveAll(backup); err != nil {
		return fmt.Errorf("failed to clean backup path: %w", err)
	}

	if err := os.Rename(current, backup); err != nil {
		return fmt.Errorf("failed to backup current installation: %w", err)
	}

	if err := os.Rename(candidatePath, current); err != nil {
		rollbackErr := os.Rename(backup, current)
		if rollbackErr != nil {
			return fmt.Errorf("failed to apply update and rollback failed: %v; rollback error: %v", err, rollbackErr)
		}
		return fmt.Errorf("failed to apply update: %w", err)
	}

	if err := os.RemoveAll(backup); err != nil {
		return fmt.Errorf("update applied but failed to remove backup: %w", err)
	}

	return nil
}

func printUpdateProgress(read, total int64) {
	if total <= 0 {
		fmt.Printf("\r[INFO] Downloaded %s", formatBytes(read))
		return
	}

	percent := int(float64(read) * 100 / float64(total))
	if percent > 100 {
		percent = 100
	}
	if percent < 0 {
		percent = 0
	}

	const barWidth = 10
	filled := percent * barWidth / 100
	if filled > barWidth {
		filled = barWidth
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	fmt.Printf("\r[%s] %d%% (%s / %s)", bar, percent, formatBytes(read), formatBytes(total))
}

func formatBytes(size int64) string {
	if size < 1024 {
		return strconv.FormatInt(size, 10) + "B"
	}

	kb := float64(size) / 1024
	if kb < 1024 {
		return fmt.Sprintf("%.1fKB", kb)
	}

	mb := kb / 1024
	if mb < 1024 {
		return fmt.Sprintf("%.1fMB", mb)
	}

	gb := mb / 1024
	return fmt.Sprintf("%.1fGB", gb)
}
