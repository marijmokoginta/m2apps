package daemon

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"m2apps/internal/api"
	"m2apps/internal/progress"
	"m2apps/internal/storage"
	"m2apps/internal/system"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

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
	return nil
}

func (m *Manager) Start() error {
	if err := m.Install(); err != nil {
		return err
	}

	running, _ := m.isRunning()
	if running {
		return nil
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	cmd := system.NewProcessCommand(execPath, "daemon", "run")
	devNull, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0o644)
	if devNull != nil {
		defer devNull.Close()
		cmd.Stdout = devNull
		cmd.Stderr = devNull
	}
	cmd.Env = append(os.Environ(), "M2APPS_DAEMON=1")

	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	}

	if err := cmd.Start(); err != nil {
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

	return fmt.Errorf("daemon start timed out")
}

func (m *Manager) Stop() error {
	pid, err := m.readPID()
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	process, err := os.FindProcess(pid)
	if err == nil {
		_ = process.Kill()
	}

	_ = os.Remove(m.pidFile)
	_ = os.Remove(m.portFile)
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
		return err
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to start daemon listener: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	if err := m.writeRuntimeFiles(os.Getpid(), port); err != nil {
		return err
	}
	defer m.cleanupRuntimeFiles()

	server := api.NewServer(store, progress.DefaultManager())

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		_ = server.Shutdown()
		return nil
	case err := <-errCh:
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
	if err := os.WriteFile(m.appsFile, raw, 0o600); err != nil {
		return fmt.Errorf("failed to write daemon app registry: %w", err)
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
	if err := os.WriteFile(m.pidFile, []byte(strconv.Itoa(pid)), 0o600); err != nil {
		return fmt.Errorf("failed to write daemon pid file: %w", err)
	}
	if err := os.WriteFile(m.portFile, []byte(strconv.Itoa(port)), 0o600); err != nil {
		return fmt.Errorf("failed to write daemon port file: %w", err)
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

	process, err := os.FindProcess(pid)
	if err != nil {
		return false, nil
	}

	err = process.Signal(syscall.Signal(0))
	if err == nil {
		return true, nil
	}

	if runtime.GOOS == "windows" && strings.Contains(strings.ToLower(err.Error()), "not supported") {
		return true, nil
	}

	return false, nil
}
