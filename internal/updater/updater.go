package updater

import (
	"archive/zip"
	"fmt"
	"m2apps/internal/downloader"
	"m2apps/internal/github"
	"m2apps/internal/installer"
	"m2apps/internal/preset"
	"m2apps/internal/process"
	"m2apps/internal/progress"
	"m2apps/internal/storage"
	"m2apps/internal/system"
	"m2apps/internal/ui"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
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

	stageRoot := filepath.Join(system.GetBaseDir(), "update_stage", config.AppID)
	if err := os.MkdirAll(stageRoot, 0o755); err != nil {
		pm.Log(id, err.Error())
		pm.Fail(id)
		return fmt.Errorf("failed to create update stage directory: %w", err)
	}

	downloadPath := filepath.Join(stageRoot, "downloads", asset.Name)
	if err := os.MkdirAll(filepath.Dir(downloadPath), 0o755); err != nil {
		pm.Log(id, err.Error())
		pm.Fail(id)
		return fmt.Errorf("failed to create download directory: %w", err)
	}

	shouldDownload, err := shouldDownloadUpdateArchive(downloadPath, 30*24*time.Hour)
	if err != nil {
		pm.Log(id, err.Error())
		pm.Fail(id)
		return err
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
	if shouldDownload {
		if err := dl.Download(asset.URL, downloadPath, downloadProgress); err != nil {
			pm.Log(id, err.Error())
			pm.Fail(id)
			return err
		}
	} else {
		pm.Log(id, fmt.Sprintf("Reusing cached archive %s", filepath.Base(downloadPath)))
	}
	fmt.Println()
	fmt.Println(ui.Success("[OK] Update package downloaded"))
	pm.Update(id, "install", "running installer", 65)
	pm.Log(id, "Update package downloaded")

	processManager, err := process.NewManager()
	if err != nil {
		pm.Log(id, err.Error())
		pm.Fail(id)
		return fmt.Errorf("failed to initialize process manager: %w", err)
	}

	candidateDir := filepath.Join(stageRoot, "candidate")
	if err := os.RemoveAll(candidateDir); err != nil && !os.IsNotExist(err) {
		pm.Log(id, err.Error())
		pm.Fail(id)
		return fmt.Errorf("failed to clean update candidate directory: %w", err)
	}
	defer os.RemoveAll(candidateDir)
	installCtx := installer.InstallContext{
		ZipPath:         downloadPath,
		TargetDir:       candidateDir,
		Preset:          config.Preset,
		AppID:           config.AppID,
		ProgressManager: pm,
		ProgressAppID:   id,
		Mode:            installer.ModeUpdate,
	}

	if err := installer.Install(installCtx); err != nil {
		pm.Log(id, err.Error())
		pm.Fail(id)
		return fmt.Errorf("failed to install update package: %w", err)
	}

	beforeStatus, err := processManager.Status(config.AppID)
	if err != nil {
		pm.Log(id, err.Error())
		pm.Fail(id)
		return fmt.Errorf("failed to check app process status before update: %w", err)
	}
	runningBefore := runningProcessNames(beforeStatus.Processes)

	if len(runningBefore) > 0 {
		pm.Update(id, "preflight", "stopping app processes", 85)
		pm.Log(id, fmt.Sprintf("Stopping app processes: %s", strings.Join(runningBefore, ", ")))
		fmt.Println(ui.Info(fmt.Sprintf("[INFO] Stopping app %s before applying update...", config.AppID)))
		if _, err := processManager.Stop(config.AppID); err != nil {
			pm.Log(id, err.Error())
			pm.Fail(id)
			return fmt.Errorf("failed to stop app processes before applying update: %w", err)
		}
		if runtime.GOOS == "windows" {
			time.Sleep(700 * time.Millisecond)
		}
	}

	pm.Update(id, "apply", "applying update", 90)
	pm.Log(id, "Applying update to install path")
	if err := replaceInstall(config.AppID, candidateDir, config.InstallPath); err != nil {
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

	if _, err := processManager.SyncAppURL(config.AppID); err != nil {
		pm.Log(id, err.Error())
		pm.Fail(id)
		return fmt.Errorf("failed to sync APP_URL after update: %w", err)
	}

	restartTargets := computeRestartTargets(config.Preset, runningBefore)
	if len(restartTargets) > 0 {
		if _, err := processManager.RestartNamed(config.AppID, restartTargets...); err != nil {
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

func runningProcessNames(processes []process.Process) []string {
	if len(processes) == 0 {
		return nil
	}

	out := make([]string, 0)
	seen := make(map[string]struct{})
	for _, proc := range processes {
		if !strings.EqualFold(strings.TrimSpace(proc.Status), "running") {
			continue
		}
		name := strings.TrimSpace(proc.Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, name)
	}
	return out
}

func computeRestartTargets(presetName string, runningBefore []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0)

	for _, name := range runningBefore {
		key := strings.ToLower(strings.TrimSpace(name))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, strings.TrimSpace(name))
	}

	// If the app was running before update, also ensure preset critical targets are restarted.
	if len(out) > 0 {
		for _, name := range preset.RestartProcessTargets(presetName) {
			key := strings.ToLower(strings.TrimSpace(name))
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, strings.TrimSpace(name))
		}
	}

	return out
}

func shouldDownloadUpdateArchive(path string, maxAge time.Duration) (bool, error) {
	target := strings.TrimSpace(path)
	if target == "" {
		return true, fmt.Errorf("update archive path is empty")
	}
	if maxAge <= 0 {
		maxAge = 30 * 24 * time.Hour
	}

	info, err := os.Stat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, fmt.Errorf("failed to check archive cache %s: %w", target, err)
	}
	if info.IsDir() {
		return false, fmt.Errorf("archive cache path is a directory: %s", target)
	}
	if info.Size() <= 0 {
		return true, nil
	}

	age := time.Since(info.ModTime())
	if age > maxAge {
		return true, nil
	}

	ok, err := isValidZipArchive(target)
	if err != nil {
		return false, err
	}
	if !ok {
		_ = os.Remove(target)
		return true, nil
	}

	return false, nil
}

func isValidZipArchive(path string) (bool, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "not a valid zip file") ||
			strings.Contains(lower, "zip: not a valid zip file") ||
			strings.Contains(lower, "unexpected eof") {
			return false, nil
		}
		return false, fmt.Errorf("failed to validate archive %s: %w", path, err)
	}
	defer reader.Close()

	if len(reader.File) == 0 {
		return false, nil
	}
	return true, nil
}

func replaceInstall(appID string, candidatePath string, installPath string) error {
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

	if err := renameWithRetry(current, backup); err != nil {
		if isWindowsDirBusyError(err) {
			return fmt.Errorf("failed to backup current installation: %w\n\n%s", err, windowsLockHelp(appID, current))
		}
		return fmt.Errorf("failed to backup current installation: %w", err)
	}

	if err := renameWithRetry(candidatePath, current); err != nil {
		rollbackErr := renameWithRetry(backup, current)
		if rollbackErr != nil {
			return fmt.Errorf("failed to apply update and rollback failed: %v; rollback error: %v", err, rollbackErr)
		}
		if isWindowsDirBusyError(err) {
			return fmt.Errorf("failed to apply update: %w\n\n%s", err, windowsLockHelp(appID, current))
		}
		return fmt.Errorf("failed to apply update: %w", err)
	}

	if err := os.RemoveAll(backup); err != nil {
		return fmt.Errorf("update applied but failed to remove backup: %w", err)
	}

	return nil
}

func renameWithRetry(from, to string) error {
	src := filepath.Clean(strings.TrimSpace(from))
	dst := filepath.Clean(strings.TrimSpace(to))
	if src == "" || dst == "" {
		return fmt.Errorf("invalid rename path")
	}

	if runtime.GOOS != "windows" {
		return os.Rename(src, dst)
	}

	// Windows may keep handles for a short time after process termination or due to AV scans.
	const attempts = 8
	backoff := 200 * time.Millisecond
	for i := 0; i < attempts; i++ {
		err := os.Rename(src, dst)
		if err == nil {
			return nil
		}
		if !isWindowsDirBusyError(err) {
			return err
		}
		time.Sleep(backoff)
		if backoff < 2*time.Second {
			backoff *= 2
		}
	}

	return fmt.Errorf("rename failed after retries (source=%s, target=%s)", src, dst)
}

func isWindowsDirBusyError(err error) bool {
	if err == nil || runtime.GOOS != "windows" {
		return false
	}

	text := strings.ToLower(strings.TrimSpace(err.Error()))
	if text == "" {
		return false
	}

	return strings.Contains(text, "being used by another process") ||
		strings.Contains(text, "used by another process") ||
		strings.Contains(text, "access is denied") ||
		strings.Contains(text, "permission denied")
}

func windowsLockHelp(appID string, installPath string) string {
	id := strings.TrimSpace(appID)
	if id == "" {
		id = "<app_id>"
	}
	path := strings.TrimSpace(installPath)
	if path == "" {
		path = "<app_path>"
	}

	return fmt.Sprintf(
		"Windows file-lock detected. To fix:\n- Stop app processes: m2apps app stop %s\n- Close terminals/editors using: %s\n- Retry update after a few seconds\n",
		id,
		path,
	)
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
