package preset

import (
	"fmt"
	"m2apps/internal/logger"
	"m2apps/internal/ui"
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

		logWriter := logger.Writer()
		if logWriter == nil {
			return fmt.Errorf("logger is not initialized")
		}

		if _, err := logWriter.WriteString(fmt.Sprintf("\n=== Step [%d/%d]: %s ===\n", i+1, total, commandLine)); err != nil {
			return fmt.Errorf("failed to write step log: %w", err)
		}

		cmd := buildShellCommand(commandLine)
		cmd.Dir = workDir
		cmd.Stdout = logWriter
		cmd.Stderr = logWriter
		cmd.Env = append(os.Environ(),
			"CI=true",
			"NPM_CONFIG_LOGLEVEL=silent",
			"NO_COLOR=1",
		)

		spinner := ui.NewSpinner()
		spinner.Start(fmt.Sprintf("[%d/%d] Running: %s", i+1, total, commandLine))

		if err := cmd.Run(); err != nil {
			spinner.Stop(ui.Error(fmt.Sprintf("[FAIL] %s", commandLine)))
			return fmt.Errorf("step failed: %s (see logs)", commandLine)
		}

		spinner.Stop(ui.Success(fmt.Sprintf("[OK] %s", commandLine)))
	}

	return nil
}

func buildShellCommand(command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/C", command)
	}
	return exec.Command("sh", "-c", command)
}
