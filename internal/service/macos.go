package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	macOSLaunchAgentLabel = "com.m2apps.daemon"
)

type MacOSService struct{}

func NewMacOSService() ServiceManager {
	return &MacOSService{}
}

func (s *MacOSService) Install() error {
	if err := validateBinary(unixBinaryPath); err != nil {
		return err
	}

	plistPath, err := macOSPlistPath()
	if err != nil {
		return err
	}
	if exists, err := fileExists(plistPath); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("service already exists at %s", plistPath)
	}

	if err := os.MkdirAll(filepath.Dir(plistPath), 0755); err != nil {
		return fmt.Errorf("failed to create launch agent directory: %w", err)
	}
	if err := os.WriteFile(plistPath, []byte(macOSPlistContent(unixBinaryPath)), 0644); err != nil {
		return fmt.Errorf("failed to write launch agent plist: %w", err)
	}

	return runCommand("launchctl", "load", plistPath)
}

func (s *MacOSService) Uninstall() error {
	plistPath, err := macOSPlistPath()
	if err != nil {
		return err
	}
	if exists, err := fileExists(plistPath); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("service not found")
	}

	_ = runCommand("launchctl", "unload", plistPath)
	if err := os.Remove(plistPath); err != nil {
		return fmt.Errorf("failed to remove launch agent plist: %w", err)
	}
	return nil
}

func (s *MacOSService) Enable() error {
	plistPath, err := macOSPlistPath()
	if err != nil {
		return err
	}
	if exists, err := fileExists(plistPath); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("service not found")
	}
	return runCommand("launchctl", "load", plistPath)
}

func (s *MacOSService) Disable() error {
	plistPath, err := macOSPlistPath()
	if err != nil {
		return err
	}
	if exists, err := fileExists(plistPath); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("service not found")
	}
	return runCommand("launchctl", "unload", plistPath)
}

func (s *MacOSService) Start() error {
	if _, err := s.Status(); err != nil {
		return err
	}
	return runCommand("launchctl", "start", macOSLaunchAgentLabel)
}

func (s *MacOSService) Stop() error {
	if _, err := s.Status(); err != nil {
		return err
	}
	return runCommand("launchctl", "stop", macOSLaunchAgentLabel)
}

func (s *MacOSService) Status() (string, error) {
	plistPath, err := macOSPlistPath()
	if err != nil {
		return "", err
	}
	if exists, err := fileExists(plistPath); err != nil {
		return "", err
	} else if !exists {
		return "", fmt.Errorf("service not found")
	}

	output, err := runCommandOutput("launchctl", "list", macOSLaunchAgentLabel)
	if err != nil {
		if strings.Contains(strings.ToLower(output), "could not find service") {
			return "stopped", nil
		}
		return "", fmt.Errorf("failed to query service status: %w", err)
	}

	if strings.TrimSpace(output) == "" {
		return "stopped", nil
	}
	return "running", nil
}

func macOSPlistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve user home: %w", err)
	}
	return filepath.Join(home, "Library", "LaunchAgents", "com.m2apps.daemon.plist"), nil
}

func macOSPlistContent(binaryPath string) string {
	return strings.Join([]string{
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>",
		"<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">",
		"<plist version=\"1.0\">",
		"<dict>",
		"    <key>Label</key>",
		"    <string>com.m2apps.daemon</string>",
		"",
		"    <key>ProgramArguments</key>",
		"    <array>",
		fmt.Sprintf("        <string>%s</string>", binaryPath),
		"        <string>daemon</string>",
		"        <string>run</string>",
		"    </array>",
		"",
		"    <key>RunAtLoad</key>",
		"    <true/>",
		"</dict>",
		"</plist>",
		"",
	}, "\n")
}
