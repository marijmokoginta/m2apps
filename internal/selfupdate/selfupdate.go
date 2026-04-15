package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"m2apps/internal/downloader"
	"m2apps/internal/github"
	"m2apps/internal/system"
	"m2apps/internal/ui"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	releaseOwner = "marijmokoginta"
	releaseRepo  = "m2apps"
)

var ErrRestartScheduled = errors.New("self-update restart scheduled")

type CheckResult struct {
	CurrentVersion string
	LatestVersion  string
	HasUpdate      bool
	Skipped        bool
}

type skipState struct {
	SkippedVersion string `json:"skipped_version"`
}

func Check(currentVersion string) (CheckResult, error) {
	current := strings.TrimSpace(currentVersion)
	if current == "" {
		return CheckResult{}, fmt.Errorf("current version is required")
	}

	client := github.NewClient("")
	release, err := client.GetLatestRelease(releaseOwner, releaseRepo)
	if err != nil {
		return CheckResult{}, err
	}

	cmp, err := github.CompareVersionTags(release.TagName, current)
	if err != nil {
		return CheckResult{}, fmt.Errorf("failed to compare versions: %w", err)
	}

	skippedVersion, err := ReadSkippedVersion()
	if err != nil {
		return CheckResult{}, err
	}

	hasUpdate := cmp > 0
	return CheckResult{
		CurrentVersion: current,
		LatestVersion:  release.TagName,
		HasUpdate:      hasUpdate,
		Skipped:        hasUpdate && skippedVersion == release.TagName,
	}, nil
}

func Update(currentVersion string) error {
	current := strings.TrimSpace(currentVersion)
	if current == "" {
		return fmt.Errorf("current version is required")
	}

	client := github.NewClient("")
	release, err := client.GetLatestRelease(releaseOwner, releaseRepo)
	if err != nil {
		return err
	}

	cmp, err := github.CompareVersionTags(release.TagName, current)
	if err != nil {
		return fmt.Errorf("failed to compare versions: %w", err)
	}
	if cmp <= 0 {
		return nil
	}

	assetName, err := assetNameForCurrentPlatform()
	if err != nil {
		return err
	}

	asset, err := github.FindAsset(release, assetName)
	if err != nil {
		return fmt.Errorf("failed to find release asset %q: %w", assetName, err)
	}

	archivePath := filepath.Join(os.TempDir(), asset.Name)
	if err := os.Remove(archivePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean temporary archive file: %w", err)
	}

	dl := downloader.New("")
	fmt.Println(ui.Info(fmt.Sprintf("[INFO] Downloading update package: %s", asset.Name)))
	downloadProgress := func(read, total int64) {
		printSelfUpdateDownloadProgress(read, total)
	}
	if err := dl.Download(asset.URL, archivePath, downloadProgress); err != nil {
		fmt.Println()
		return err
	}
	fmt.Println()
	fmt.Println(ui.Success("[OK] Update package downloaded"))

	newBinaryPath := filepath.Join(os.TempDir(), "m2apps_new"+executableSuffix())
	installSpinner := ui.NewSpinner()
	installSpinner.Start("[INFO] Installing self-update...")
	stopInstallSpinner := func(message string) {
		installSpinner.Stop(message)
	}

	if err := extractBinaryFromArchive(archivePath, newBinaryPath); err != nil {
		stopInstallSpinner(ui.Error(fmt.Sprintf("[FAIL] Install update failed: %v", err)))
		return err
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(newBinaryPath, 0o755); err != nil {
			stopInstallSpinner(ui.Error(fmt.Sprintf("[FAIL] Install update failed: %v", err)))
			return fmt.Errorf("failed to make updated binary executable: %w", err)
		}
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	if runtime.GOOS == "windows" {
		if err := launchWindowsUpdater(execPath, newBinaryPath); err != nil {
			stopInstallSpinner(ui.Error(fmt.Sprintf("[FAIL] Install update failed: %v", err)))
			return err
		}
		stopInstallSpinner(ui.Success("[OK] Install update completed"))
		_ = SaveSkippedVersion("")
		return ErrRestartScheduled
	}

	if err := applyAndRestart(execPath, newBinaryPath); err != nil {
		stopInstallSpinner(ui.Error(fmt.Sprintf("[FAIL] Install update failed: %v", err)))
		return err
	}
	stopInstallSpinner(ui.Success("[OK] Install update completed"))
	_ = SaveSkippedVersion("")
	return ErrRestartScheduled
}

func RunInternalSelfUpdate(targetPath, newBinaryPath string, parentPID int) error {
	target := filepath.Clean(strings.TrimSpace(targetPath))
	newBin := filepath.Clean(strings.TrimSpace(newBinaryPath))
	if target == "" {
		return fmt.Errorf("target path is required")
	}
	if newBin == "" {
		return fmt.Errorf("new binary path is required")
	}

	if parentPID > 0 {
		waitForProcessExit(parentPID, 45*time.Second)
	}

	if _, err := os.Stat(newBin); err != nil {
		return fmt.Errorf("new binary not found: %w", err)
	}

	if err := replaceBinary(target, newBin); err != nil {
		return err
	}

	return restartBinary(target)
}

func ReadSkippedVersion() (string, error) {
	state, err := readSkipState()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(state.SkippedVersion), nil
}

func SaveSkippedVersion(version string) error {
	state := skipState{
		SkippedVersion: strings.TrimSpace(version),
	}
	return writeSkipState(state)
}

func readSkipState() (skipState, error) {
	path := filepath.Join(system.GetBaseDir(), "self_update.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return skipState{}, nil
		}
		return skipState{}, fmt.Errorf("failed to read self-update state: %w", err)
	}
	if len(data) == 0 {
		return skipState{}, nil
	}

	var state skipState
	if err := json.Unmarshal(data, &state); err != nil {
		return skipState{}, fmt.Errorf("failed to parse self-update state: %w", err)
	}
	return state, nil
}

func writeSkipState(state skipState) error {
	if err := os.MkdirAll(system.GetBaseDir(), 0o755); err != nil {
		return fmt.Errorf("failed to prepare self-update directory: %w", err)
	}

	path := filepath.Join(system.GetBaseDir(), "self_update.json")
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize self-update state: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return fmt.Errorf("failed to write self-update state: %w", err)
	}
	return nil
}

func assetNameForCurrentPlatform() (string, error) {
	if runtime.GOARCH != "amd64" {
		return "", fmt.Errorf("self-update is not supported for architecture %s", runtime.GOARCH)
	}

	switch runtime.GOOS {
	case "windows":
		return "m2apps-windows-amd64.zip", nil
	case "linux":
		return "m2apps-linux-amd64.tar.gz", nil
	case "darwin":
		return "m2apps-darwin-amd64.tar.gz", nil
	default:
		return "", fmt.Errorf("self-update is not supported on %s", runtime.GOOS)
	}
}

func executableSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func archiveBinaryName() string {
	if runtime.GOOS == "windows" {
		return "m2apps.exe"
	}
	return "m2apps"
}

func extractBinaryFromArchive(archivePath, targetBinaryPath string) error {
	name := strings.ToLower(filepath.Base(archivePath))
	switch {
	case strings.HasSuffix(name, ".zip"):
		return extractBinaryFromZip(archivePath, targetBinaryPath)
	case strings.HasSuffix(name, ".tar.gz"):
		return extractBinaryFromTarGz(archivePath, targetBinaryPath)
	default:
		return fmt.Errorf("unsupported self-update archive format: %s", archivePath)
	}
}

func extractBinaryFromZip(archivePath, targetBinaryPath string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open update archive: %w", err)
	}
	defer reader.Close()

	expected := archiveBinaryName()
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if filepath.Base(file.Name) != expected {
			continue
		}

		src, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to read update binary from archive: %w", err)
		}
		defer src.Close()

		return writeBinaryFile(targetBinaryPath, src, file.Mode())
	}

	return fmt.Errorf("binary %s not found in update archive", expected)
}

func extractBinaryFromTarGz(archivePath, targetBinaryPath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open update archive: %w", err)
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to read update gzip stream: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	expected := archiveBinaryName()
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read update tar stream: %w", err)
		}
		if header.FileInfo().IsDir() {
			continue
		}
		if filepath.Base(header.Name) != expected {
			continue
		}

		return writeBinaryFile(targetBinaryPath, tr, header.FileInfo().Mode())
	}

	return fmt.Errorf("binary %s not found in update archive", expected)
}

func writeBinaryFile(path string, src io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to prepare update output directory: %w", err)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean previous update output: %w", err)
	}

	perm := mode.Perm()
	if perm == 0 {
		if runtime.GOOS == "windows" {
			perm = 0o644
		} else {
			perm = 0o755
		}
	}

	dst, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return fmt.Errorf("failed to create update binary: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to write update binary: %w", err)
	}
	return nil
}

func launchWindowsUpdater(targetPath, newBinaryPath string) error {
	helperPath := filepath.Join(os.TempDir(), fmt.Sprintf("m2apps_updater_%d.exe", time.Now().UnixNano()))
	if err := copyFile(targetPath, helperPath, 0o755); err != nil {
		return fmt.Errorf("failed to prepare windows updater helper: %w", err)
	}

	cmd := exec.Command(
		helperPath,
		"internal",
		"self-update",
		"--target", targetPath,
		"--new", newBinaryPath,
		"--parent-pid", strconv.Itoa(os.Getpid()),
	)
	configureUpdaterProcess(cmd)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start windows updater helper: %w", err)
	}
	return nil
}

func applyAndRestart(targetPath, newBinaryPath string) error {
	if err := replaceBinary(targetPath, newBinaryPath); err != nil {
		return err
	}
	return restartBinary(targetPath)
}

func replaceBinary(targetPath, newBinaryPath string) error {
	target := filepath.Clean(targetPath)
	newBin := filepath.Clean(newBinaryPath)
	backup := target + "_old"

	if _, err := os.Stat(target); err != nil {
		return fmt.Errorf("current binary not found: %w", err)
	}

	if err := os.Remove(backup); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean previous backup binary: %w", err)
	}

	if err := os.Rename(target, backup); err != nil {
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	if err := moveFile(newBin, target); err != nil {
		_ = os.Rename(backup, target)
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(target, 0o755); err != nil {
			return fmt.Errorf("binary replaced but failed to set executable permission: %w", err)
		}
	}

	return nil
}

func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	} else if !isCrossDeviceErr(err) {
		return err
	}

	if err := copyFile(src, dst, 0o755); err != nil {
		return err
	}
	if err := os.Remove(src); err != nil {
		return fmt.Errorf("failed to clean temporary file %s: %w", src, err)
	}
	return nil
}

func isCrossDeviceErr(err error) bool {
	var linkErr *os.LinkError
	if !errors.As(err, &linkErr) {
		return false
	}
	return errors.Is(linkErr.Err, syscall.EXDEV)
}

func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", src, err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("failed to prepare directory for %s: %w", dst, err)
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", dst, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("failed to copy %s to %s: %w", src, dst, err)
	}
	return nil
}

func restartBinary(executablePath string) error {
	cmd := exec.Command(executablePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	configureUpdaterProcess(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to restart application: %w", err)
	}
	return nil
}

func waitForProcessExit(pid int, timeout time.Duration) {
	if pid <= 0 {
		return
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !isPIDRunning(pid) {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func isPIDRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	if runtime.GOOS == "windows" {
		out, err := system.CombinedOutput("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid))
		if err != nil {
			return false
		}
		text := strings.ToLower(string(out))
		if strings.Contains(text, "no tasks are running") {
			return false
		}
		return strings.Contains(text, strconv.Itoa(pid))
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func printSelfUpdateDownloadProgress(read, total int64) {
	if total <= 0 {
		fmt.Printf("\r[INFO] Downloaded %s", formatSelfUpdateBytes(read))
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

	bar := strings.Repeat("=", filled) + strings.Repeat("-", barWidth-filled)
	fmt.Printf("\r[%s] %d%% (%s / %s)", bar, percent, formatSelfUpdateBytes(read), formatSelfUpdateBytes(total))
}

func formatSelfUpdateBytes(size int64) string {
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
