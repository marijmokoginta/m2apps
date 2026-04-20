package updater

import (
	"fmt"
	"m2apps/internal/downloader"
	"m2apps/internal/github"
	"m2apps/internal/installer"
	"m2apps/internal/preset"
	"m2apps/internal/process"
	"m2apps/internal/progress"
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

	pm := progress.DefaultManager()
	pm.Start(id)
	pm.Update(id, "metadata", "loading metadata", 5)
	pm.Log(id, "Loading app metadata")

	store, err := storage.New()
	if err != nil {
		pm.Log(id, err.Error())
		pm.Fail(id)
		return err
	}

	config, err := store.Load(id)
	if err != nil {
		pm.Log(id, err.Error())
		pm.Fail(id)
		return fmt.Errorf("failed to load app metadata: %w", err)
	}

	channel := github.NormalizeChannel(config.Channel)
	fmt.Println(ui.Info(fmt.Sprintf("[INFO] Checking update (channel: %s)...", channel)))
	pm.Update(id, "release", "resolving release", 15)
	pm.Log(id, fmt.Sprintf("Resolving release for channel %s", channel))

	owner, repo, err := github.ParseRepo(config.Repo)
	if err != nil {
		pm.Log(id, err.Error())
		pm.Fail(id)
		return err
	}

	client := github.NewClient(config.Token)
	target, err := github.SelectLatestReleaseByChannel(client, owner, repo, channel)
	if err != nil {
		pm.Log(id, err.Error())
		pm.Fail(id)
		return err
	}
	fmt.Println(ui.Info(fmt.Sprintf("[INFO] Current: %s", config.Version)))
	fmt.Println(ui.Info(fmt.Sprintf("[INFO] Latest:  %s", target.TagName)))

	newer, err := IsNewer(target.TagName, config.Version)
	if err != nil {
		pm.Log(id, err.Error())
		pm.Fail(id)
		return fmt.Errorf("failed to compare versions: %w", err)
	}
	if !newer {
		fmt.Println(ui.Success("[OK] Already up to date"))
		pm.Update(id, "complete", "already up to date", 100)
		pm.Complete(id)
		return nil
	}

	asset, err := github.FindAssetByVersionedName(target, config.Asset)
	if err != nil {
		pm.Log(id, err.Error())
		pm.Fail(id)
		return fmt.Errorf("Asset not found in selected release")
	}

	stageRoot := filepath.Join(filepath.Dir(config.InstallPath), ".m2apps_update_stage", config.AppID)
	if err := os.RemoveAll(stageRoot); err != nil {
		pm.Log(id, err.Error())
		pm.Fail(id)
		return fmt.Errorf("failed to clean update stage directory: %w", err)
	}
	if err := os.MkdirAll(stageRoot, 0o755); err != nil {
		pm.Log(id, err.Error())
		pm.Fail(id)
		return fmt.Errorf("failed to create update stage directory: %w", err)
	}
	defer os.RemoveAll(stageRoot)

	downloadPath := filepath.Join(stageRoot, "downloads", asset.Name)
	if err := os.MkdirAll(filepath.Dir(downloadPath), 0o755); err != nil {
		pm.Log(id, err.Error())
		pm.Fail(id)
		return fmt.Errorf("failed to create download directory: %w", err)
	}

	fmt.Println(ui.Info("[INFO] Downloading update..."))
	pm.Update(id, "download", "downloading update package", 30)
	pm.Log(id, fmt.Sprintf("Downloading %s", asset.Name))
	dl := downloader.New(config.Token)
	downloadProgress := func(read, total int64) {
		printUpdateProgress(read, total)
		if total > 0 {
			percent := int(float64(read) * 100 / float64(total))
			if percent > 100 {
				percent = 100
			}
			pm.Update(id, "download", "downloading update package", 30+(percent*30/100))
		}
	}
	if err := dl.Download(asset.URL, downloadPath, downloadProgress); err != nil {
		pm.Log(id, err.Error())
		pm.Fail(id)
		return err
	}
	fmt.Println()
	fmt.Println(ui.Success("[OK] Update package downloaded"))
	pm.Update(id, "install", "running installer", 65)
	pm.Log(id, "Update package downloaded")

	candidateDir := filepath.Join(stageRoot, "candidate")
	installCtx := installer.InstallContext{
		ZipPath:         downloadPath,
		TargetDir:       candidateDir,
		Preset:          config.Preset,
		AppID:           config.AppID,
		ProgressManager: pm,
		ProgressAppID:   id,
	}

	if err := installer.Install(installCtx); err != nil {
		pm.Log(id, err.Error())
		pm.Fail(id)
		return fmt.Errorf("failed to install update package: %w", err)
	}

	pm.Update(id, "apply", "applying update", 90)
	pm.Log(id, "Applying update to install path")
	if err := replaceInstall(candidateDir, config.InstallPath); err != nil {
		pm.Log(id, err.Error())
		pm.Fail(id)
		return err
	}
	cleanupUpdateArtifacts(config.InstallPath, downloadPath)

	if err := preset.RunPostUpdate(config.Preset, config.InstallPath); err != nil {
		pm.Log(id, err.Error())
		pm.Fail(id)
		return fmt.Errorf("failed to run post-update preset tasks: %w", err)
	}

	processManager, err := process.NewManager()
	if err != nil {
		pm.Log(id, err.Error())
		pm.Fail(id)
		return fmt.Errorf("failed to initialize process manager: %w", err)
	}

	if _, err := processManager.SyncAppURL(config.AppID); err != nil {
		pm.Log(id, err.Error())
		pm.Fail(id)
		return fmt.Errorf("failed to sync APP_URL after update: %w", err)
	}

	targets := preset.RestartProcessTargets(config.Preset)
	if len(targets) > 0 {
		if _, err := processManager.RestartNamed(config.AppID, targets...); err != nil {
			pm.Log(id, err.Error())
			pm.Fail(id)
			return fmt.Errorf("failed to restart preset processes: %w", err)
		}
	}

	config.Version = target.TagName
	config.Channel = channel
	if err := store.Save(config.AppID, config); err != nil {
		pm.Log(id, err.Error())
		pm.Fail(id)
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	pm.Update(id, "complete", "update completed", 100)
	pm.Log(id, "Update completed")
	pm.Complete(id)
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

func cleanupUpdateArtifacts(installPath string, downloadPath string) {
	if strings.TrimSpace(downloadPath) != "" {
		if err := os.Remove(downloadPath); err != nil && !os.IsNotExist(err) {
			fmt.Println(ui.Warning(fmt.Sprintf("[WARN] Failed to remove downloaded update package: %v", err)))
		}
	}

	tmpRoot := filepath.Join(filepath.Clean(strings.TrimSpace(installPath)), ".m2apps_tmp")
	if err := os.RemoveAll(tmpRoot); err != nil && !os.IsNotExist(err) {
		fmt.Println(ui.Warning(fmt.Sprintf("[WARN] Failed to clean temp directory %s: %v", tmpRoot, err)))
	}
}
