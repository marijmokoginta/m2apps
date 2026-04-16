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
	if isElevatedViaPowerShell() {
		return true
	}

	// Common admin-required command on Windows.
	cmd := system.NewCommand("fltmc")
	if cmd.Run() == nil {
		return true
	}

	// Legacy fallback for older environments.
	cmd = system.NewCommand("net", "session")
	return cmd.Run() == nil
}

func isElevatedViaPowerShell() bool {
	script := "$id=[Security.Principal.WindowsIdentity]::GetCurrent();$p=New-Object Security.Principal.WindowsPrincipal($id);if($p.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)){Write-Output 'true'}else{Write-Output 'false'}"
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).CombinedOutput()
	if err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(string(out)), "true")
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
