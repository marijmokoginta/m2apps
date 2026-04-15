package service

import (
	"errors"
	"fmt"
	"m2apps/internal/system"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	linuxServiceName = "m2apps"
	linuxServiceFile = "/etc/systemd/system/m2apps.service"
)

type LinuxService struct{}

func NewLinuxService() ServiceManager {
	return &LinuxService{}
}

func (s *LinuxService) Install() error {
	if err := requireRoot(); err != nil {
		return err
	}
	execPath, err := resolveExecutablePath()
	if err != nil {
		return err
	}
	if err := validateBinary(execPath); err != nil {
		return err
	}
	if exists, err := fileExists(linuxServiceFile); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("service already exists at %s", linuxServiceFile)
	}

	baseDir := filepath.Clean(system.GetBaseDir())
	runUser, runGroup := resolveServiceRuntimeAccount()
	if err := ensureServiceRuntimeOwnership(baseDir, runUser, runGroup); err != nil {
		return err
	}
	unit := linuxServiceUnit(execPath, baseDir, runUser, runGroup)
	if err := os.WriteFile(linuxServiceFile, []byte(unit), 0644); err != nil {
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
	baseDir := filepath.Clean(system.GetBaseDir())
	runUser, runGroup := resolveServiceRuntimeAccount()
	if err := ensureServiceRuntimeOwnership(baseDir, runUser, runGroup); err != nil {
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

func resolveServiceRuntimeAccount() (string, string) {
	if name := strings.TrimSpace(os.Getenv("SUDO_USER")); name != "" && !strings.EqualFold(name, "root") {
		return name, name
	}

	if rawUID := strings.TrimSpace(os.Getenv("PKEXEC_UID")); rawUID != "" {
		if _, err := strconv.Atoi(rawUID); err == nil {
			if u, err := user.LookupId(rawUID); err == nil {
				groupName := strings.TrimSpace(u.Username)
				if grp, err := user.LookupGroupId(u.Gid); err == nil && strings.TrimSpace(grp.Name) != "" {
					groupName = strings.TrimSpace(grp.Name)
				}
				return strings.TrimSpace(u.Username), groupName
			}
		}
	}

	if name := strings.TrimSpace(os.Getenv("USER")); name != "" && !strings.EqualFold(name, "root") {
		return name, name
	}

	return "", ""
}

func ensureServiceRuntimeOwnership(baseDir, runUser, runGroup string) error {
	if strings.TrimSpace(runUser) == "" {
		return nil
	}

	u, err := user.Lookup(runUser)
	if err != nil {
		return fmt.Errorf("failed to resolve service user %s: %w", runUser, err)
	}

	uid, err := strconv.Atoi(strings.TrimSpace(u.Uid))
	if err != nil {
		return fmt.Errorf("failed to parse uid for %s: %w", runUser, err)
	}

	targetGroup := strings.TrimSpace(runGroup)
	if targetGroup == "" {
		targetGroup = strings.TrimSpace(u.Username)
	}
	grp, err := user.LookupGroup(targetGroup)
	if err != nil {
		return fmt.Errorf("failed to resolve service group %s: %w", targetGroup, err)
	}

	gid, err := strconv.Atoi(strings.TrimSpace(grp.Gid))
	if err != nil {
		return fmt.Errorf("failed to parse gid for %s: %w", targetGroup, err)
	}

	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return fmt.Errorf("failed to create base directory %s: %w", baseDir, err)
	}

	if err := filepath.WalkDir(baseDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := os.Chown(path, uid, gid); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to update ownership under %s: %w", baseDir, err)
	}

	return nil
}

func linuxServiceUnit(binaryPath, baseDir, runUser, runGroup string) string {
	lines := []string{
		"[Unit]",
		"Description=M2Apps Daemon",
		"After=network.target",
		"",
		"[Service]",
		fmt.Sprintf("Environment=M2APPS_HOME=%s", baseDir),
		fmt.Sprintf("ExecStart=%s daemon run", binaryPath),
		"Restart=always",
	}
	if strings.TrimSpace(runUser) != "" {
		lines = append(lines, fmt.Sprintf("User=%s", runUser))
	}
	if strings.TrimSpace(runGroup) != "" {
		lines = append(lines, fmt.Sprintf("Group=%s", runGroup))
	}
	lines = append(lines,
		"",
		"[Install]",
		"WantedBy=multi-user.target",
		"",
	)
	return strings.Join(lines, "\n")
}
