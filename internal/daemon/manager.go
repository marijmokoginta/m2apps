package daemon

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"m2apps/internal/api"
	"m2apps/internal/env"
	procman "m2apps/internal/process"
	"m2apps/internal/progress"
	"m2apps/internal/storage"
	"m2apps/internal/system"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type AppProcess struct {
	AppID string
	Name  string
	PID   int
}

type Manager struct {
	baseDir  string
	pidFile  string
	portFile string
	appsFile string
}

func NewManager() (*Manager, error) {
	base := system.GetDaemonDir()
	return &Manager{
		baseDir:  base,
		pidFile:  filepath.Join(base, "daemon.pid"),
		portFile: filepath.Join(base, "daemon.port"),
		appsFile: filepath.Join(base, "apps.json"),
	}, nil
}

func (m *Manager) Install() error {
	if err := os.MkdirAll(m.baseDir, 0o755); err != nil {
		return fmt.Errorf("failed to create daemon directory: %w", err)
	}
	if err := os.Chmod(m.baseDir, 0o755); err != nil && !os.IsPermission(err) {
		return fmt.Errorf("failed to set daemon directory permission: %w", err)
	}
	if err := os.MkdirAll(system.GetLogDir(), 0o755); err != nil {
		return fmt.Errorf("failed to create daemon log directory: %w", err)
	}
	if err := os.Chmod(system.GetLogDir(), 0o755); err != nil && !os.IsPermission(err) {
		return fmt.Errorf("failed to set daemon log directory permission: %w", err)
	}
	return nil
}

func (m *Manager) Start() error {
	if err := m.Install(); err != nil {
		return err
	}

	running, _ := m.isRunning()
	if running {
		if m.isAPIReachable() {
			return nil
		}

		m.logErrorf("stale daemon runtime detected (pid exists but API is not reachable), restarting daemon process")
		_ = os.Remove(m.pidFile)
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	logFile, err := m.openDaemonLogFile()
	if err != nil {
		return err
	}
	defer logFile.Close()

	cmd := system.NewProcessCommand(execPath, "daemon", "run")
	cmd.Stdin = nil
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Env = append(os.Environ(), "M2APPS_DAEMON=1")
	configureDaemonProcess(cmd)

	if err := cmd.Start(); err != nil {
		m.logErrorf("failed to start daemon process: %v", err)
		return fmt.Errorf("failed to start daemon process: %w", err)
	}
	_ = cmd.Process.Release()

	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := m.Port(); err == nil {
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}

	m.logErrorf("daemon start timed out waiting for runtime files")
	return fmt.Errorf("daemon start timed out (check log: %s)", ServiceLogPath())
}

func (m *Manager) Stop() error {
	pid, err := m.readPID()
	if err != nil {
		if os.IsNotExist(err) {
			m.cleanupRuntimeFiles()
			return nil
		}
		return err
	}

	if err := terminatePID(pid); err != nil && daemonProcessAlive(pid) {
		return err
	}

	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if !daemonProcessAlive(pid) {
			m.cleanupRuntimeFiles()
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}

	if daemonProcessAlive(pid) {
		return fmt.Errorf("failed to stop daemon pid %d: process is still running", pid)
	}

	m.cleanupRuntimeFiles()
	return nil
}

func (m *Manager) Status() (string, error) {
	running, err := m.isRunning()
	if err != nil {
		return "", err
	}
	if !running {
		return "stopped", nil
	}

	port, err := m.Port()
	if err != nil {
		return "running", nil
	}

	return fmt.Sprintf("running (port: %d)", port), nil
}

func (m *Manager) RunForeground(ctx context.Context) error {
	if err := m.Install(); err != nil {
		return err
	}

	store, err := storage.New()
	if err != nil {
		m.logErrorf("failed to initialize storage: %v", err)
		return err
	}

	listener, port, err := m.bindListener()
	if err != nil {
		m.logErrorf("failed to bind daemon listener: %v", err)
		return err
	}
	defer listener.Close()

	if err := m.writeRuntimeFiles(os.Getpid(), port); err != nil {
		m.logErrorf("failed to write daemon runtime files: %v", err)
		return err
	}
	defer m.cleanupRuntimeFiles()

	m.logInfof("daemon started pid=%d port=%d", os.Getpid(), port)
	m.syncAppAPIEnv(store, port)
	m.autoStartApps(store)

	server := api.NewServer(store, progress.DefaultManager(), func(message string) {
		_ = AppendAccessLog(message)
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		_ = server.Shutdown()
		m.logInfof("daemon shutdown requested")
		return nil
	case err := <-errCh:
		if err != nil {
			m.logErrorf("daemon runtime error: %v", err)
		}
		return err
	}
}

func (m *Manager) Port() (int, error) {
	data, err := os.ReadFile(m.portFile)
	if err != nil {
		return 0, fmt.Errorf("failed to read daemon port: %w", err)
	}

	port, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid daemon port file")
	}

	return port, nil
}

func (m *Manager) RegisterApp(appID string) error {
	id := strings.TrimSpace(appID)
	if id == "" {
		return fmt.Errorf("app_id is required")
	}

	if err := m.Install(); err != nil {
		return err
	}

	apps := []string{}
	if data, err := os.ReadFile(m.appsFile); err == nil {
		_ = json.Unmarshal(data, &apps)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read daemon app registry: %w", err)
	}

	for _, existing := range apps {
		if existing == id {
			return nil
		}
	}

	apps = append(apps, id)
	raw, err := json.MarshalIndent(apps, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize app registry: %w", err)
	}
	if err := os.WriteFile(m.appsFile, raw, 0o644); err != nil {
		return fmt.Errorf("failed to write daemon app registry: %w", err)
	}
	if err := os.Chmod(m.appsFile, 0o644); err != nil && !os.IsPermission(err) {
		return fmt.Errorf("failed to set daemon app registry permission: %w", err)
	}

	return nil
}

func (m *Manager) UnregisterApp(appID string) error {
	id := strings.TrimSpace(appID)
	if id == "" {
		return fmt.Errorf("app_id is required")
	}

	if err := m.Install(); err != nil {
		return err
	}

	apps := []string{}
	if data, err := os.ReadFile(m.appsFile); err == nil {
		_ = json.Unmarshal(data, &apps)
	} else if os.IsNotExist(err) {
		return nil
	} else {
		return fmt.Errorf("failed to read daemon app registry: %w", err)
	}

	filtered := make([]string, 0, len(apps))
	for _, existing := range apps {
		if existing != id {
			filtered = append(filtered, existing)
		}
	}

	raw, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize app registry: %w", err)
	}
	if err := os.WriteFile(m.appsFile, raw, 0o644); err != nil {
		return fmt.Errorf("failed to write daemon app registry: %w", err)
	}
	if err := os.Chmod(m.appsFile, 0o644); err != nil && !os.IsPermission(err) {
		return fmt.Errorf("failed to set daemon app registry permission: %w", err)
	}

	return nil
}

func GenerateAPIToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to generate API token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func (m *Manager) writeRuntimeFiles(pid, port int) error {
	if err := os.WriteFile(m.pidFile, []byte(strconv.Itoa(pid)), 0o644); err != nil {
		return fmt.Errorf("failed to write daemon pid file: %w", err)
	}
	if err := os.Chmod(m.pidFile, 0o644); err != nil && !os.IsPermission(err) {
		return fmt.Errorf("failed to set daemon pid file permission: %w", err)
	}
	if err := os.WriteFile(m.portFile, []byte(strconv.Itoa(port)), 0o644); err != nil {
		return fmt.Errorf("failed to write daemon port file: %w", err)
	}
	if err := os.Chmod(m.portFile, 0o644); err != nil && !os.IsPermission(err) {
		return fmt.Errorf("failed to set daemon port file permission: %w", err)
	}
	return nil
}

func (m *Manager) cleanupRuntimeFiles() {
	_ = os.Remove(m.pidFile)
	_ = os.Remove(m.portFile)
}

func (m *Manager) readPID() (int, error) {
	data, err := os.ReadFile(m.pidFile)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid daemon pid file")
	}
	return pid, nil
}

func (m *Manager) isRunning() (bool, error) {
	pid, err := m.readPID()
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return daemonProcessAlive(pid), nil
}

func terminatePID(pid int) error {
	if pid <= 0 {
		return nil
	}

	pidValue := strconv.Itoa(pid)
	if runtime.GOOS == "windows" {
		out, err := system.CombinedOutput("taskkill", "/PID", pidValue, "/T", "/F")
		if err != nil {
			lower := strings.ToLower(strings.TrimSpace(string(out)))
			if strings.Contains(lower, "not found") || strings.Contains(lower, "no running instance") {
				return nil
			}
			return fmt.Errorf("failed to stop daemon pid %d: %s", pid, strings.TrimSpace(string(out)))
		}
		return nil
	}

	out, err := system.CombinedOutput("kill", "-TERM", pidValue)
	if err != nil && daemonProcessAlive(pid) {
		out, err = system.CombinedOutput("kill", "-KILL", pidValue)
		if err != nil && daemonProcessAlive(pid) {
			return fmt.Errorf("failed to stop daemon pid %d: %s", pid, strings.TrimSpace(string(out)))
		}
	}

	return nil
}

func daemonProcessAlive(pid int) bool {
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

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}

	lower := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(lower, "operation not permitted")
}

func (m *Manager) bindListener() (net.Listener, int, error) {
	if savedPort, ok := m.readSavedPort(); ok {
		addr := fmt.Sprintf("127.0.0.1:%d", savedPort)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			if isAddressInUse(err) {
				return nil, 0, fmt.Errorf("daemon port %d is already in use", savedPort)
			}
			return nil, 0, fmt.Errorf("failed to bind daemon listener on %s: %w", addr, err)
		}
		return listener, savedPort, nil
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to start daemon listener: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	return listener, port, nil
}

func (m *Manager) readSavedPort() (int, bool) {
	raw, err := os.ReadFile(m.portFile)
	if err != nil {
		return 0, false
	}
	port, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || port <= 0 {
		return 0, false
	}
	return port, true
}

func isAddressInUse(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "address already in use") ||
		strings.Contains(message, "only one usage of each socket address")
}

func (m *Manager) openDaemonLogFile() (*os.File, error) {
	if err := os.MkdirAll(system.GetLogDir(), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create daemon log directory: %w", err)
	}
	logFile, err := os.OpenFile(ServiceLogPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open daemon service log file: %w", err)
	}
	return logFile, nil
}

func (m *Manager) logInfof(format string, args ...any) {
	m.logWithLevel("INFO", format, args...)
}

func (m *Manager) logErrorf(format string, args ...any) {
	m.logWithLevel("ERROR", format, args...)
}

func (m *Manager) logWithLevel(level, format string, args ...any) {
	prefix := strings.ToUpper(strings.TrimSpace(level))
	if prefix == "" {
		prefix = "INFO"
	}
	message := strings.TrimSpace(fmt.Sprintf(format, args...))
	if message == "" {
		return
	}

	_ = AppendServiceLog(prefix, message)
}

func (m *Manager) autoStartApps(store storage.Storage) {
	cfgs, err := m.loadInstalledApps(store)
	if err != nil {
		m.logErrorf("failed to load installed apps: %v", err)
		return
	}

	pm, err := procman.NewManager()
	if err != nil {
		m.logErrorf("failed to initialize process manager: %v", err)
		return
	}

	started := make([]AppProcess, 0)
	for _, cfg := range cfgs {
		if !cfg.AutoStart {
			m.logInfof("auto-start disabled for app %s", cfg.AppID)
			continue
		}

		status, err := pm.Status(cfg.AppID)
		if err == nil && hasRunningProcess(status.Processes) {
			m.logInfof("skip auto-start for app %s because process is already running", cfg.AppID)
			continue
		}

		result, err := pm.Start(cfg.AppID)
		if err != nil {
			lower := strings.ToLower(err.Error())
			if strings.Contains(lower, "already running") {
				m.logInfof("skip auto-start for app %s because process is already running", cfg.AppID)
				continue
			}
			m.logErrorf("failed to auto-start app %s: %v", cfg.AppID, err)
			continue
		}

		for _, proc := range result.Processes {
			started = append(started, AppProcess{
				AppID: cfg.AppID,
				Name:  strings.TrimSpace(proc.Name),
				PID:   proc.PID,
			})
		}
	}

	for _, proc := range started {
		m.logInfof("auto-started app=%s process=%s pid=%d", proc.AppID, proc.Name, proc.PID)
	}
}

func (m *Manager) loadInstalledApps(store storage.Storage) ([]storage.AppConfig, error) {
	entries, err := os.ReadDir(system.GetAppsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return []storage.AppConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read apps directory: %w", err)
	}

	appIDs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := strings.TrimSpace(entry.Name())
		if id == "" {
			continue
		}
		appIDs = append(appIDs, id)
	}

	sort.Strings(appIDs)
	configs := make([]storage.AppConfig, 0, len(appIDs))
	for _, appID := range appIDs {
		cfg, err := store.Load(appID)
		if err != nil {
			m.logErrorf("failed to load app metadata for %s: %v", appID, err)
			continue
		}
		cfg.AppID = strings.TrimSpace(cfg.AppID)
		if cfg.AppID == "" {
			cfg.AppID = appID
		}
		configs = append(configs, cfg)
	}

	return configs, nil
}

func hasRunningProcess(processes []procman.Process) bool {
	for _, proc := range processes {
		if strings.EqualFold(strings.TrimSpace(proc.Status), "running") {
			return true
		}
	}
	return false
}

func (m *Manager) isAPIReachable() bool {
	port, err := m.Port()
	if err != nil || port <= 0 {
		return false
	}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 2*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func (m *Manager) syncAppAPIEnv(store storage.Storage, port int) {
	if port <= 0 {
		return
	}

	configs, err := m.loadInstalledApps(store)
	if err != nil {
		m.logErrorf("failed to load apps for API env sync: %v", err)
		return
	}

	apiURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	for _, cfg := range configs {
		installPath := strings.TrimSpace(cfg.InstallPath)
		if installPath == "" {
			continue
		}

		vars := map[string]string{
			"M2APPS_API_URL": apiURL,
		}
		if strings.TrimSpace(cfg.APIToken) != "" {
			vars["M2APPS_API_TOKEN"] = strings.TrimSpace(cfg.APIToken)
		}
		if strings.TrimSpace(cfg.AppID) != "" {
			vars["M2APPS_APP_ID"] = strings.TrimSpace(cfg.AppID)
		}

		if err := env.Upsert(installPath, vars); err != nil {
			m.logErrorf("failed to sync API env for app %s: %v", cfg.AppID, err)
			continue
		}

		m.logInfof("synced API env for app %s with %s", cfg.AppID, apiURL)
	}
}
