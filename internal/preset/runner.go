package preset

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func RunSteps(steps []Step, workDir string) error {
	total := len(steps)

	for i, step := range steps {
		switch strings.ToLower(strings.TrimSpace(step.Type)) {
		case "command":
		default:
			return fmt.Errorf("unsupported step type %q", step.Type)
		}

		commandLine := strings.TrimSpace(step.Run)
		if commandLine == "" {
			return fmt.Errorf("empty command in step %d", i+1)
		}

		fmt.Printf("[%d/%d] %s\n", i+1, total, commandLine)

		cmd := buildShellCommand(commandLine)
		cmd.Dir = workDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("installation failed at step: %s", commandLine)
		}
	}

	return nil
}

func buildShellCommand(command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/C", command)
	}
	return exec.Command("sh", "-c", command)
}
