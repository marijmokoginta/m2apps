package process

import (
	"fmt"
	"m2apps/internal/env"
	"m2apps/internal/network"
	appruntime "m2apps/internal/runtime"
	"m2apps/internal/storage"
	"m2apps/internal/system"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type Manager struct {
	store    storage.Storage
	registry *Registry
}

func NewManager() (*Manager, error) {
	store, err := storage.New()
	if err != nil {
		return nil, err
	}

	return &Manager{
		store:    store,
		registry: NewRegistry(),
	}, nil
}

func (m *Manager) Start(appID string) (AppProcesses, error) {
	id := strings.TrimSpace(appID)
	if id == "" {
		return AppProcesses{}, fmt.Errorf("app_id is required")
	}

	cfg, err := m.store.Load(id)
	if err != nil {
		return AppProcesses{}, fmt.Errorf("failed to load app metadata: %w", err)
	}

	workDir := filepath.Clean(strings.TrimSpace(cfg.InstallPath))
	if workDir == "" || workDir == "." {
		return AppProcesses{}, fmt.Errorf("invalid install path for app %s", id)
	}

	if stat, err := os.Stat(workDir); err != nil || !stat.IsDir() {
		return AppProcesses{}, fmt.Errorf("install path not found for app %s", id)
	}

	processDefs, err := appruntime.LoadPreset(cfg.Preset)
	if err != nil {
		return AppProcesses{}, err
	}

	current, err := m.Status(id)
	if err != nil {
		return AppProcesses{}, err
	}
	for _, proc := range current.Processes {
		if strings.EqualFold(strings.TrimSpace(proc.Status), "running") {
			return AppProcesses{}, fmt.Errorf("application %s is already running", id)
		}
	}

	resolvedPort := m.resolveRuntimePort(cfg.Preset, current.Processes)

	logFile, err := openAppLogFile(id)
	if err != nil {
		return AppProcesses{}, err
	}
	defer logFile.Close()

	started := make([]Process, 0, len(processDefs))
	for _, def := range processDefs {
		command := applyPortPlaceholder(def.Command, resolvedPort)
		if len(command) == 0 {
			continue
		}

		cmd := system.NewProcessCommand(command[0], command[1:]...)
		cmd.Dir = workDir
		cmd.Stdout = logFile
		cmd.Stderr = logFile
		cmd.Env = os.Environ()

		if err := cmd.Start(); err != nil {
			for _, created := range started {
				_ = stopByPID(created.PID)
			}
			return AppProcesses{}, fmt.Errorf("failed to start process %s: %w", def.Name, err)
		}

		started = append(started, Process{
			Name:    strings.TrimSpace(def.Name),
			PID:     cmd.Process.Pid,
			Port:    resolvedPort,
			Command: command,
			Status:  "running",
		})

		_ = cmd.Process.Release()
	}

	if err := m.registry.Set(id, started); err != nil {
		for _, created := range started {
			_ = stopByPID(created.PID)
		}
		return AppProcesses{}, err
	}

	if err := env.InjectAppURL(workDir, resolvedPort); err != nil {
		for _, created := range started {
			_ = stopByPID(created.PID)
		}
		return AppProcesses{}, fmt.Errorf("failed to inject APP_URL into env: %w", err)
	}

	return AppProcesses{
		AppID:     id,
		Processes: started,
	}, nil
}

func (m *Manager) Stop(appID string) (AppProcesses, error) {
	id := strings.TrimSpace(appID)
	if id == "" {
		return AppProcesses{}, fmt.Errorf("app_id is required")
	}

	entry, err := m.registry.Get(id)
	if err != nil {
		return AppProcesses{}, err
	}

	if len(entry.Processes) == 0 {
		return entry, nil
	}

	updated := make([]Process, 0, len(entry.Processes))
	for _, process := range entry.Processes {
		current := process

		if current.PID <= 0 {
			current.Status = "stopped"
			updated = append(updated, current)
			continue
		}

		if err := stopByPID(current.PID); err != nil {
			if isProcessAlive(current.PID) {
				return AppProcesses{}, err
			}
		}

		if isProcessAlive(current.PID) {
			current.Status = "running"
		} else {
			current.Status = "stopped"
		}

		updated = append(updated, current)
	}

	if err := m.registry.Set(id, updated); err != nil {
		return AppProcesses{}, err
	}

	return AppProcesses{
		AppID:     id,
		Processes: updated,
	}, nil
}

func (m *Manager) Restart(appID string) (AppProcesses, error) {
	id := strings.TrimSpace(appID)
	if id == "" {
		return AppProcesses{}, fmt.Errorf("app_id is required")
	}

	if _, err := m.Stop(id); err != nil {
		return AppProcesses{}, err
	}

	return m.Start(id)
}

func (m *Manager) Status(appID string) (AppProcesses, error) {
	id := strings.TrimSpace(appID)
	if id == "" {
		return AppProcesses{}, fmt.Errorf("app_id is required")
	}

	entry, err := m.registry.Get(id)
	if err != nil {
		return AppProcesses{}, err
	}

	if len(entry.Processes) == 0 {
		return entry, nil
	}

	changed := false
	updated := make([]Process, 0, len(entry.Processes))

	for _, process := range entry.Processes {
		current := process
		prevStatus := strings.ToLower(strings.TrimSpace(current.Status))
		newStatus := "stopped"

		if current.PID > 0 && isProcessAlive(current.PID) {
			newStatus = "running"
		}

		current.Status = newStatus
		if prevStatus != newStatus {
			changed = true
		}

		updated = append(updated, current)
	}

	if changed {
		if err := m.registry.Set(id, updated); err != nil {
			return AppProcesses{}, err
		}
	}

	return AppProcesses{
		AppID:     id,
		Processes: updated,
	}, nil
}

func openAppLogFile(appID string) (*os.File, error) {
	if err := os.MkdirAll(system.GetLogDir(), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	logPath := filepath.Join(system.GetLogDir(), fmt.Sprintf("%s.log", appID))
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open app log file: %w", err)
	}

	return file, nil
}

func stopByPID(pid int) error {
	if pid <= 0 {
		return nil
	}

	pidValue := strconv.Itoa(pid)
	if runtime.GOOS == "windows" {
		out, err := system.CombinedOutput("taskkill", "/PID", pidValue, "/T", "/F")
		if err != nil {
			if !isProcessAlive(pid) {
				return nil
			}
			return fmt.Errorf("failed to stop pid %d: %s", pid, strings.TrimSpace(string(out)))
		}
		return nil
	}

	out, err := system.CombinedOutput("kill", "-TERM", pidValue)
	if err != nil && isProcessAlive(pid) {
		out, err = system.CombinedOutput("kill", "-KILL", pidValue)
		if err != nil && isProcessAlive(pid) {
			return fmt.Errorf("failed to stop pid %d: %s", pid, strings.TrimSpace(string(out)))
		}
	}

	return nil
}

func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	pidValue := strconv.Itoa(pid)
	if runtime.GOOS == "windows" {
		out, err := system.CombinedOutput("tasklist", "/FI", fmt.Sprintf("PID eq %s", pidValue))
		if err != nil {
			return false
		}

		text := strings.ToLower(string(out))
		if strings.Contains(text, "no tasks are running") || strings.Contains(text, "info: no tasks") {
			return false
		}
		return strings.Contains(text, pidValue)
	}

	_, err := system.CombinedOutput("kill", "-0", pidValue)
	return err == nil
}

func (m *Manager) resolveRuntimePort(preset string, existing []Process) int {
	basePort := appruntime.DefaultPort(preset)
	if basePort <= 0 {
		return 0
	}

	storedPort := findStoredPort(existing)
	if storedPort > 0 && network.IsPortAvailable(storedPort) {
		return storedPort
	}

	return network.ResolvePort(basePort)
}

func findStoredPort(processes []Process) int {
	for _, process := range processes {
		if process.Port > 0 {
			return process.Port
		}
	}
	return 0
}

func applyPortPlaceholder(command []string, port int) []string {
	resolved := make([]string, 0, len(command))
	for _, part := range command {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		if port > 0 {
			item = strings.ReplaceAll(item, "{PORT}", strconv.Itoa(port))
		}
		resolved = append(resolved, item)
	}
	return resolved
}
