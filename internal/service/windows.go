package service

import (
	"fmt"
	"m2apps/internal/privilege"
	"strings"
)

const (
	windowsServiceName = "M2Apps"
)

type WindowsService struct{}

func NewWindowsService() ServiceManager {
	return &WindowsService{}
}

func (s *WindowsService) Install() error {
	if err := requireAdmin(); err != nil {
		return err
	}
	execPath, err := resolveExecutablePath()
	if err != nil {
		return err
	}
	if err := validateBinary(execPath); err != nil {
		return err
	}

	status, err := s.Status()
	if err == nil && status != "" {
		return fmt.Errorf("service already exists")
	}
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "service not found") {
		return err
	}

	binPathValue := fmt.Sprintf("\"%s\" daemon run", execPath)
	return runSC("create", windowsServiceName, "binPath=", binPathValue, "start=", "auto")
}

func (s *WindowsService) Uninstall() error {
	if err := requireAdmin(); err != nil {
		return err
	}
	if _, err := s.Status(); err != nil {
		return err
	}

	stopErr := runSC("stop", windowsServiceName)
	if stopErr != nil && !isServiceAlreadyStopped(stopErr.Error()) {
		return stopErr
	}
	return runSC("delete", windowsServiceName)
}

func (s *WindowsService) Enable() error {
	if err := requireAdmin(); err != nil {
		return err
	}
	if _, err := s.Status(); err != nil {
		return err
	}
	return runSC("config", windowsServiceName, "start=", "auto")
}

func (s *WindowsService) Disable() error {
	if err := requireAdmin(); err != nil {
		return err
	}
	if _, err := s.Status(); err != nil {
		return err
	}
	return runSC("config", windowsServiceName, "start=", "demand")
}

func (s *WindowsService) Start() error {
	if err := requireAdmin(); err != nil {
		return err
	}
	if _, err := s.Status(); err != nil {
		return err
	}

	err := runSC("start", windowsServiceName)
	if err != nil && !isServiceAlreadyRunning(err.Error()) {
		return err
	}
	return nil
}

func (s *WindowsService) Stop() error {
	if err := requireAdmin(); err != nil {
		return err
	}
	if _, err := s.Status(); err != nil {
		return err
	}

	err := runSC("stop", windowsServiceName)
	if err != nil && !isServiceAlreadyStopped(err.Error()) {
		return err
	}
	return nil
}

func (s *WindowsService) Status() (string, error) {
	output, err := runSCOutput("query", windowsServiceName)
	if err != nil {
		if isServiceNotFound(output) {
			return "", fmt.Errorf("service not found")
		}
		return "", err
	}

	upper := strings.ToUpper(output)
	switch {
	case strings.Contains(upper, "RUNNING"):
		return "running", nil
	case strings.Contains(upper, "STOPPED"):
		return "stopped", nil
	case strings.Contains(upper, "START_PENDING"):
		return "start_pending", nil
	case strings.Contains(upper, "STOP_PENDING"):
		return "stop_pending", nil
	default:
		return "unknown", nil
	}
}

func requireAdmin() error {
	if !isAdmin() {
		return fmt.Errorf("[ERROR] Administrator privileges required. Please run terminal as Administrator.")
	}
	return nil
}

func isAdmin() bool {
	return privilege.IsElevated()
}

func runSC(args ...string) error {
	_, err := runSCOutput(args...)
	return err
}

func runSCOutput(args ...string) (string, error) {
	output, err := runCommandOutput("sc", args...)
	if err != nil {
		return output, fmt.Errorf("sc %s failed: %w", strings.Join(args, " "), err)
	}
	return output, nil
}

func isServiceNotFound(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "failed 1060") ||
		strings.Contains(lower, "does not exist as an installed service")
}

func isServiceAlreadyRunning(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "already running")
}

func isServiceAlreadyStopped(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "has not been started") ||
		strings.Contains(lower, "service has not been started")
}
