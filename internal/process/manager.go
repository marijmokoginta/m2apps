package process

import (
	"fmt"
	"m2apps/internal/env"
	"m2apps/internal/hostmode"
	"m2apps/internal/network"
	"m2apps/internal/preset"
	appruntime "m2apps/internal/runtime"
	"m2apps/internal/storage"
	"m2apps/internal/system"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
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
	runtimeHost, appURLHost, err := resolveAppHosts(cfg.Preset, cfg.ServerMode)
	if err != nil {
		return AppProcesses{}, err
	}

	if err := maybeInjectSanctumStatefulDomains(workDir, cfg.Preset, resolvedPort); err != nil {
		return AppProcesses{}, err
	}

	logFile, err := openAppLogFile(id)
	if err != nil {
		return AppProcesses{}, err
	}
	defer logFile.Close()

	started := make([]Process, 0, len(processDefs))
	for _, def := range processDefs {
		command := applyPortPlaceholder(withProcessHost(def.Name, def.Command, runtimeHost), resolvedPort)
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

	appURL := fmt.Sprintf("http://%s:%d", appURLHost, resolvedPort)
	appURLChanged, err := env.UpsertWithResult(workDir, map[string]string{
		"APP_URL": appURL,
	})
	if err != nil {
		for _, created := range started {
			_ = stopByPID(created.PID)
		}
		return AppProcesses{}, fmt.Errorf("failed to inject APP_URL into env: %w", err)
	}

	if appURLChanged {
		if err := preset.RunOnAppURLChange(cfg.Preset, workDir); err != nil {
			for _, created := range started {
				_ = stopByPID(created.PID)
			}
			return AppProcesses{}, fmt.Errorf("failed to run preset tasks on APP_URL change: %w", err)
		}
	}

	return AppProcesses{
		AppID:     id,
		Processes: started,
	}, nil
}

func (m *Manager) SyncAppURL(appID string) (bool, error) {
	id := strings.TrimSpace(appID)
	if id == "" {
		return false, fmt.Errorf("app_id is required")
	}

	cfg, err := m.store.Load(id)
	if err != nil {
		return false, fmt.Errorf("failed to load app metadata: %w", err)
	}

	current, err := m.Status(id)
	if err != nil {
		return false, err
	}

	resolvedPort := m.resolveRuntimePort(cfg.Preset, current.Processes)
	_, appURLHost, err := resolveAppHosts(cfg.Preset, cfg.ServerMode)
	if err != nil {
		return false, err
	}

	workDir := filepath.Clean(strings.TrimSpace(cfg.InstallPath))
	appURL := fmt.Sprintf("http://%s:%d", appURLHost, resolvedPort)

	changed, err := env.UpsertWithResult(workDir, map[string]string{
		"APP_URL": appURL,
	})
	if err != nil {
		return false, fmt.Errorf("failed to inject APP_URL into env: %w", err)
	}

	if changed {
		if err := preset.RunOnAppURLChange(cfg.Preset, workDir); err != nil {
			return true, fmt.Errorf("failed to run preset tasks on APP_URL change: %w", err)
		}
	}

	return changed, nil
}

func maybeInjectSanctumStatefulDomains(workDir string, presetName string, port int) error {
	if !isLaravelPresetName(presetName) {
		return nil
	}

	ip, err := network.ResolveLocalIPv4()
	if err != nil || strings.TrimSpace(ip) == "" {
		// LAN IP may not exist (offline / no active interface). Don't block app start.
		return nil
	}

	additions := []string{ip}
	if port > 0 {
		additions = append(additions, fmt.Sprintf("%s:%d", ip, port))
	}

	updated, err := appendEnvCSVUnique(workDir, "SANCTUM_STATEFUL_DOMAINS", additions...)
	if err != nil {
		return fmt.Errorf("failed to inject SANCTUM_STATEFUL_DOMAINS: %w", err)
	}
	_ = updated
	return nil
}

func appendEnvCSVUnique(workDir string, key string, additions ...string) (bool, error) {
	key = strings.TrimSpace(key)
	if key == "" || len(additions) == 0 {
		return false, nil
	}

	current, err := env.ReadValues(workDir, []string{key})
	if err != nil {
		return false, err
	}

	raw := strings.TrimSpace(current[key])
	quote := byte(0)
	if len(raw) >= 2 && ((raw[0] == '"' && raw[len(raw)-1] == '"') || (raw[0] == '\'' && raw[len(raw)-1] == '\'')) {
		quote = raw[0]
		raw = raw[1 : len(raw)-1]
		raw = strings.TrimSpace(raw)
	}

	items := make([]string, 0)
	seen := make(map[string]struct{})
	for _, part := range strings.Split(raw, ",") {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		lower := strings.ToLower(item)
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		items = append(items, item)
	}

	changed := false
	for _, add := range additions {
		add = strings.TrimSpace(add)
		if add == "" {
			continue
		}
		lower := strings.ToLower(add)
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		items = append(items, add)
		changed = true
	}

	if !changed {
		return false, nil
	}

	value := strings.Join(items, ", ")
	if quote != 0 {
		value = string(quote) + value + string(quote)
	}

	if err := env.Upsert(workDir, map[string]string{key: value}); err != nil {
		return false, err
	}
	return true, nil
}

func (m *Manager) RestartNamed(appID string, processNames ...string) (AppProcesses, error) {
	id := strings.TrimSpace(appID)
	if id == "" {
		return AppProcesses{}, fmt.Errorf("app_id is required")
	}

	if len(processNames) == 0 {
		return m.Status(id)
	}

	entry, err := m.registry.Get(id)
	if err != nil {
		return AppProcesses{}, err
	}
	if len(entry.Processes) == 0 {
		return entry, nil
	}

	cfg, err := m.store.Load(id)
	if err != nil {
		return AppProcesses{}, fmt.Errorf("failed to load app metadata: %w", err)
	}

	workDir := filepath.Clean(strings.TrimSpace(cfg.InstallPath))
	if workDir == "" || workDir == "." {
		return AppProcesses{}, fmt.Errorf("invalid install path for app %s", id)
	}

	logFile, err := openAppLogFile(id)
	if err != nil {
		return AppProcesses{}, err
	}
	defer logFile.Close()

	targets := make(map[string]struct{}, len(processNames))
	for _, name := range processNames {
		normalized := strings.ToLower(strings.TrimSpace(name))
		if normalized != "" {
			targets[normalized] = struct{}{}
		}
	}

	updated := make([]Process, 0, len(entry.Processes))
	for _, proc := range entry.Processes {
		current := proc
		if _, ok := targets[strings.ToLower(strings.TrimSpace(current.Name))]; !ok {
			updated = append(updated, current)
			continue
		}

		if current.PID > 0 {
			_ = stopByPID(current.PID)
		}

		if len(current.Command) == 0 {
			current.Status = "stopped"
			current.PID = 0
			updated = append(updated, current)
			continue
		}

		cmd := system.NewProcessCommand(current.Command[0], current.Command[1:]...)
		cmd.Dir = workDir
		cmd.Stdout = logFile
		cmd.Stderr = logFile
		cmd.Env = os.Environ()

		if err := cmd.Start(); err != nil {
			return AppProcesses{}, fmt.Errorf("failed to restart process %s: %w", current.Name, err)
		}

		current.PID = cmd.Process.Pid
		current.Status = "running"
		_ = cmd.Process.Release()
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
	beforeStartTime := ""
	if runtime.GOOS == "linux" {
		beforeStartTime = linuxProcStartTime(pid)
	}

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
			return fmt.Errorf("failed to stop pid %d: %s", pid, formatStopError(out, err))
		}
	}
	waitUntilStopped(pid, 2*time.Second)
	if isProcessAlive(pid) {
		if runtime.GOOS == "linux" && beforeStartTime != "" {
			afterStartTime := linuxProcStartTime(pid)
			if afterStartTime == "" || afterStartTime != beforeStartTime {
				return nil
			}
		}
		return fmt.Errorf("failed to stop pid %d: process is still running (possible permission issue or auto-restart)", pid)
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
	if err == nil {
		if runtime.GOOS == "linux" && isLinuxZombie(pid) {
			return false
		}
		return true
	}

	lower := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(lower, "operation not permitted")
}

func waitUntilStopped(pid int, timeout time.Duration) {
	if pid <= 0 {
		return
	}
	if timeout <= 0 {
		timeout = 1 * time.Second
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !isProcessAlive(pid) {
			return
		}
		time.Sleep(120 * time.Millisecond)
	}
}

func formatStopError(output []byte, cmdErr error) string {
	out := strings.TrimSpace(string(output))
	errText := ""
	if cmdErr != nil {
		errText = strings.TrimSpace(cmdErr.Error())
	}

	switch {
	case out != "" && errText != "":
		return out + " (" + errText + ")"
	case out != "":
		return out
	case errText != "":
		return errText
	default:
		return "unknown stop error"
	}
}

func isLinuxZombie(pid int) bool {
	state := linuxProcState(pid)
	return state == "Z"
}

func linuxProcStartTime(pid int) string {
	fields, ok := readLinuxProcStatFields(pid)
	if !ok || len(fields) < 22 {
		return ""
	}
	return fields[21]
}

func linuxProcState(pid int) string {
	fields, ok := readLinuxProcStatFields(pid)
	if !ok || len(fields) < 3 {
		return ""
	}
	return fields[2]
}

func readLinuxProcStatFields(pid int) ([]string, bool) {
	if pid <= 0 {
		return nil, false
	}

	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return nil, false
	}

	raw := strings.TrimSpace(string(data))
	if raw == "" {
		return nil, false
	}

	closeIdx := strings.LastIndex(raw, ")")
	firstSpace := strings.Index(raw, " ")
	if firstSpace <= 0 || closeIdx <= firstSpace || closeIdx+2 >= len(raw) {
		return nil, false
	}

	pidField := strings.TrimSpace(raw[:firstSpace])
	commField := strings.TrimSpace(raw[firstSpace+1 : closeIdx+1])
	suffix := strings.TrimSpace(raw[closeIdx+2:])
	if pidField == "" || commField == "" || suffix == "" {
		return nil, false
	}

	fields := []string{pidField, commField}
	fields = append(fields, strings.Fields(suffix)...)
	return fields, true
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

func resolveAppHosts(presetName string, mode string) (runtimeHost string, appURLHost string, err error) {
	runtimeHost = "127.0.0.1"
	appURLHost = "127.0.0.1"

	if !isLaravelPresetName(presetName) {
		return runtimeHost, appURLHost, nil
	}

	if hostmode.Normalize(mode) != hostmode.LAN {
		return runtimeHost, appURLHost, nil
	}

	ip, ipErr := network.ResolveLocalIPv4()
	if ipErr != nil {
		return "", "", fmt.Errorf("failed to resolve LAN IP for preset %s: %w", strings.TrimSpace(presetName), ipErr)
	}

	return "0.0.0.0", ip, nil
}

func withProcessHost(processName string, command []string, host string) []string {
	if strings.ToLower(strings.TrimSpace(processName)) != "web" {
		return command
	}
	if strings.TrimSpace(host) == "" {
		return command
	}

	resolved := make([]string, 0, len(command))
	replaced := false

	for i := 0; i < len(command); i++ {
		part := strings.TrimSpace(command[i])
		if part == "" {
			continue
		}

		lower := strings.ToLower(part)
		switch {
		case strings.HasPrefix(lower, "--host="):
			resolved = append(resolved, "--host="+host)
			replaced = true
		case lower == "--host":
			resolved = append(resolved, "--host")
			if i+1 < len(command) {
				resolved = append(resolved, host)
				i++
			} else {
				resolved = append(resolved, host)
			}
			replaced = true
		default:
			resolved = append(resolved, part)
		}
	}

	if !replaced {
		resolved = append(resolved, "--host="+host)
	}

	return resolved
}

func isLaravelPresetName(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "laravel", "laravel-inertia":
		return true
	default:
		return false
	}
}
