//go:build !windows

package privilege

import (
	"fmt"
	"m2apps/internal/system"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func isElevated() bool {
	return os.Geteuid() == 0
}

func relaunchElevated(args []string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	switch runtime.GOOS {
	case "linux":
		if _, err := exec.LookPath("pkexec"); err != nil {
			return fmt.Errorf("pkexec is required for privileged operation on Linux")
		}

		baseDir := strings.TrimSpace(os.Getenv("M2APPS_HOME"))
		if baseDir == "" {
			baseDir = system.GetBaseDir()
		}

		cmdArgs := []string{"env", "M2APPS_HOME=" + baseDir, execPath}
		cmdArgs = append(cmdArgs, args...)
		cmd := exec.Command("pkexec", cmdArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("privileged operation failed: %w", err)
		}
		return nil
	default:
		joined := strings.Join(append([]string{execPath}, args...), " ")
		return fmt.Errorf("privileged operation requires root. run with sudo: %s", joined)
	}
}
