package service

import (
	"errors"
	"fmt"
	"m2apps/internal/system"
	"os"
	"strings"
)

const (
	linuxServiceName = "m2apps"
	linuxServiceFile = "/etc/systemd/system/m2apps.service"
	unixBinaryPath   = "/usr/local/bin/m2apps"
)

var linuxServiceUnit = strings.Join([]string{
	"[Unit]",
	"Description=M2Apps Daemon",
	"After=network.target",
	"",
	"[Service]",
	"ExecStart=/usr/local/bin/m2apps daemon run",
	"Restart=always",
	"",
	"[Install]",
	"WantedBy=multi-user.target",
	"",
}, "\n")

type LinuxService struct{}

func NewLinuxService() ServiceManager {
	return &LinuxService{}
}

func (s *LinuxService) Install() error {
	if err := requireRoot(); err != nil {
		return err
	}
	if err := validateBinary(unixBinaryPath); err != nil {
		return err
	}
	if exists, err := fileExists(linuxServiceFile); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("service already exists at %s", linuxServiceFile)
	}

	if err := os.WriteFile(linuxServiceFile, []byte(linuxServiceUnit), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}
	if err := runCommand("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}
	return nil
}

func (s *LinuxService) Uninstall() error {
	if err := requireRoot(); err != nil {
		return err
	}
	if exists, err := fileExists(linuxServiceFile); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("service not found")
	}

	_ = runCommand("systemctl", "stop", linuxServiceName)
	_ = runCommand("systemctl", "disable", linuxServiceName)

	if err := os.Remove(linuxServiceFile); err != nil {
		return fmt.Errorf("failed to remove service file: %w", err)
	}
	if err := runCommand("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}
	return nil
}

func (s *LinuxService) Enable() error {
	if err := requireRoot(); err != nil {
		return err
	}
	if err := ensureLinuxServiceInstalled(); err != nil {
		return err
	}
	return runCommand("systemctl", "enable", linuxServiceName)
}

func (s *LinuxService) Disable() error {
	if err := requireRoot(); err != nil {
		return err
	}
	if err := ensureLinuxServiceInstalled(); err != nil {
		return err
	}
	return runCommand("systemctl", "disable", linuxServiceName)
}

func (s *LinuxService) Start() error {
	if err := requireRoot(); err != nil {
		return err
	}
	if err := ensureLinuxServiceInstalled(); err != nil {
		return err
	}
	return runCommand("systemctl", "start", linuxServiceName)
}

func (s *LinuxService) Stop() error {
	if err := requireRoot(); err != nil {
		return err
	}
	if err := ensureLinuxServiceInstalled(); err != nil {
		return err
	}
	return runCommand("systemctl", "stop", linuxServiceName)
}

func (s *LinuxService) Status() (string, error) {
	if err := ensureLinuxServiceInstalled(); err != nil {
		return "", err
	}

	output, err := runCommandOutput("systemctl", "is-active", linuxServiceName)
	state := strings.TrimSpace(output)
	if err == nil {
		return state, nil
	}

	switch state {
	case "inactive", "failed", "activating", "deactivating":
		return state, nil
	case "unknown":
		return "", fmt.Errorf("service not found")
	default:
		return "", fmt.Errorf("failed to query service status: %w", err)
	}
}

func ensureLinuxServiceInstalled() error {
	exists, err := fileExists(linuxServiceFile)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("service not found")
	}
	return nil
}

func requireRoot() error {
	if !isRootUser() {
		return fmt.Errorf("[ERROR] Root privileges required. Please run with sudo.")
	}
	return nil
}

func runCommand(cmd string, args ...string) error {
	_, err := runCommandOutput(cmd, args...)
	return err
}

func runCommandOutput(cmd string, args ...string) (string, error) {
	out, err := system.CombinedOutput(cmd, args...)
	message := strings.TrimSpace(string(out))
	if err != nil {
		if message != "" {
			return message, errors.New(message)
		}
		return "", err
	}
	return message, nil
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("failed to check path %s: %w", path, err)
}

func validateBinary(path string) error {
	exists, err := fileExists(path)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("binary not found at %s", path)
	}
	return nil
}
