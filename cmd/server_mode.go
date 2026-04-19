package cmd

import (
	"errors"
	"fmt"
	"m2apps/internal/env"
	"m2apps/internal/hostmode"
	"m2apps/internal/network"
	"m2apps/internal/preset"
	"m2apps/internal/process"
	appruntime "m2apps/internal/runtime"
	"m2apps/internal/storage"
	"m2apps/internal/ui"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var serverModeCmd = &cobra.Command{
	Use:   "server-mode",
	Short: "Manage application server mode",
}

var serverModeSetCmd = &cobra.Command{
	Use:   "set <app_id> <mode>",
	Short: "Switch server mode for an installed application",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		appID := strings.TrimSpace(args[0])
		mode := normalizeServerModeInput(args[1])
		if !confirmAction(fmt.Sprintf("Switch app %s server mode to %s?", appID, mode)) {
			fmt.Println(ui.Warning("[WARN] Server mode switch cancelled."))
			return
		}

		message, err := runSetServerMode(appID, mode)
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}
		fmt.Println(ui.Success(message))
	},
}

func init() {
	serverModeCmd.AddCommand(serverModeSetCmd)
	rootCmd.AddCommand(serverModeCmd)
}

func normalizeServerModeInput(mode string) string {
	return hostmode.Normalize(mode)
}

func isValidServerMode(mode string) bool {
	switch normalizeServerModeInput(mode) {
	case hostmode.Localhost, hostmode.LAN:
		return true
	default:
		return false
	}
}

func runSetServerMode(appID string, mode string) (string, error) {
	if !isValidServerMode(mode) {
		return "", fmt.Errorf("invalid server mode. use one of: localhost, lan")
	}

	store, err := storage.New()
	if err != nil {
		return "", err
	}

	config, err := store.Load(appID)
	if err != nil {
		return "", fmt.Errorf("failed to load app metadata: %w", err)
	}

	targetMode := normalizeServerModeInput(mode)
	currentMode := normalizeServerModeInput(config.ServerMode)
	if currentMode == targetMode {
		return fmt.Sprintf("[OK] Server mode for %s is already %s", config.AppID, targetMode), nil
	}

	presetName := strings.TrimSpace(config.Preset)
	if targetMode == hostmode.LAN && !isLaravelPresetName(presetName) {
		return "", fmt.Errorf("server_mode=lan is only supported for preset: laravel, laravel-inertia")
	}

	workDir := filepath.Clean(strings.TrimSpace(config.InstallPath))
	if workDir == "" || workDir == "." {
		return "", fmt.Errorf("invalid install path for app %s", config.AppID)
	}
	if stat, statErr := os.Stat(workDir); statErr != nil || !stat.IsDir() {
		return "", fmt.Errorf("install path not found for app %s", config.AppID)
	}

	envPath, envExisted, err := detectEnvFile(workDir)
	if err != nil {
		return "", err
	}
	prevAppURL, hadPrevAppURL, err := readEnvValue(envPath, "APP_URL")
	if err != nil {
		return "", err
	}

	manager, err := process.NewManager()
	if err != nil {
		return "", err
	}
	state, err := manager.Status(config.AppID)
	if err != nil {
		return "", err
	}

	port := resolveServerModePort(presetName, state.Processes)
	if port <= 0 {
		return "", fmt.Errorf("failed to resolve app port for %s", config.AppID)
	}

	appHost, err := resolveServerModeAppHost(presetName, targetMode)
	if err != nil {
		return "", err
	}

	previousMode := currentMode
	config.ServerMode = targetMode
	if err := store.Save(config.AppID, config); err != nil {
		return "", fmt.Errorf("failed to update server mode: %w", err)
	}

	appURLModified := false

	rollback := func(cause error) error {
		var rollbackErr error

		if appURLModified {
			if hadPrevAppURL {
				if _, err := env.UpsertWithResult(workDir, map[string]string{
					"APP_URL": prevAppURL,
				}); err != nil {
					rollbackErr = errors.Join(rollbackErr, fmt.Errorf("failed to rollback APP_URL: %w", err))
				}
			} else {
				if err := env.DeleteKeys(workDir, "APP_URL"); err != nil {
					rollbackErr = errors.Join(rollbackErr, fmt.Errorf("failed to rollback APP_URL: %w", err))
				}

				if !envExisted && strings.TrimSpace(envPath) != "" && isTriviallyEmptyEnvFile(envPath) {
					if err := os.Remove(envPath); err != nil && !os.IsNotExist(err) {
						rollbackErr = errors.Join(rollbackErr, fmt.Errorf("failed to remove generated env file: %w", err))
					}
				}
			}
		}

		config.ServerMode = previousMode
		if saveErr := store.Save(config.AppID, config); saveErr != nil {
			rollbackErr = errors.Join(rollbackErr, fmt.Errorf("failed to rollback server_mode: %w", saveErr))
		}

		if rollbackErr != nil {
			return fmt.Errorf("%v (rollback: %v)", cause, rollbackErr)
		}
		return cause
	}

	appURL := fmt.Sprintf("http://%s:%d", appHost, port)
	if _, err := env.UpsertWithResult(workDir, map[string]string{
		"APP_URL": appURL,
	}); err != nil {
		return "", rollback(fmt.Errorf("failed to inject APP_URL into env: %w", err))
	}
	appURLModified = true

	if isLaravelPresetName(presetName) {
		if err := preset.RunOnAppURLChange(presetName, workDir); err != nil {
			return "", rollback(fmt.Errorf("failed to run preset tasks on APP_URL change: %w", err))
		}
		if err := preset.RunPostUpdate(presetName, workDir); err != nil {
			return "", rollback(fmt.Errorf("failed to run preset optimize tasks: %w", err))
		}
	}

	if _, err := manager.Restart(config.AppID); err != nil {
		return "", rollback(fmt.Errorf("failed to restart app process: %w", err))
	}

	return fmt.Sprintf("[OK] Server mode for %s set to %s (%s)", config.AppID, targetMode, appURL), nil
}

func resolveServerModePort(presetName string, processes []process.Process) int {
	for _, proc := range processes {
		if proc.Port > 0 {
			return proc.Port
		}
	}

	return appruntime.DefaultPort(presetName)
}

func resolveServerModeAppHost(presetName string, mode string) (string, error) {
	if !isLaravelPresetName(presetName) {
		return "127.0.0.1", nil
	}

	if normalizeServerModeInput(mode) != hostmode.LAN {
		return "127.0.0.1", nil
	}

	ip, err := network.ResolveLocalIPv4()
	if err != nil {
		return "", fmt.Errorf("failed to resolve LAN IP for preset %s: %w", strings.TrimSpace(presetName), err)
	}
	return ip, nil
}

func detectEnvFile(installPath string) (path string, existed bool, err error) {
	candidates := []string{
		filepath.Join(installPath, ".env"),
		filepath.Join(installPath, ".env.local"),
	}

	for _, file := range candidates {
		if _, err := os.Stat(file); err == nil {
			return file, true, nil
		} else if !os.IsNotExist(err) {
			return "", false, fmt.Errorf("failed to check env file %s: %w", file, err)
		}
	}

	return filepath.Join(installPath, ".env"), false, nil
}

func readEnvValue(path string, key string) (value string, ok bool, err error) {
	target := strings.TrimSpace(path)
	if target == "" {
		return "", false, nil
	}
	content, err := os.ReadFile(target)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("failed to read env file %s: %w", target, err)
	}

	needle := strings.TrimSpace(key)
	if needle == "" {
		return "", false, nil
	}

	for _, line := range strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		if k != needle {
			continue
		}
		return strings.TrimSpace(parts[1]), true, nil
	}

	return "", false, nil
}

func isTriviallyEmptyEnvFile(path string) bool {
	target := strings.TrimSpace(path)
	if target == "" {
		return false
	}
	content, err := os.ReadFile(target)
	if err != nil {
		return false
	}

	for _, line := range strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		return false
	}
	return true
}
