//go:build windows

package privilege

import (
	"fmt"
	"m2apps/internal/system"
	"os"
	"os/exec"
	"strings"
)

func isElevated() bool {
	cmd := system.NewCommand("net", "session")
	return cmd.Run() == nil
}

func relaunchElevated(args []string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	psArgs := make([]string, 0, len(args))
	for _, arg := range args {
		psArgs = append(psArgs, "'"+escapePowerShellSingle(arg)+"'")
	}

	script := fmt.Sprintf(
		"$p = Start-Process -FilePath '%s' -ArgumentList @(%s) -Verb RunAs -Wait -PassThru; exit $p.ExitCode",
		escapePowerShellSingle(execPath),
		strings.Join(psArgs, ","),
	)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("privileged operation failed with exit code %d", exitErr.ExitCode())
		}
		return fmt.Errorf("failed to start elevated process: %w", err)
	}

	return nil
}

func escapePowerShellSingle(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
