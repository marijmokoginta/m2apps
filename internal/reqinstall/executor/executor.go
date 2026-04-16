package executor

import (
	"fmt"
	"m2apps/internal/reqinstall/plan"
	"m2apps/internal/system"
	"os"
	"strings"
)

type ExecutionResult struct {
	ToolType string
	Name     string
	Success  bool
	Skipped  bool
	Message  string
}

type Executor struct{}

func New() *Executor {
	return &Executor{}
}

func (e *Executor) Execute(candidate plan.InstallCandidate) ExecutionResult {
	result := ExecutionResult{
		ToolType: strings.ToLower(strings.TrimSpace(candidate.ToolType)),
		Name:     strings.TrimSpace(candidate.Name),
		Success:  false,
		Skipped:  false,
	}

	if len(candidate.Commands) == 0 {
		result.Message = "no install command available"
		return result
	}

	lastErr := ""
	for _, commandLine := range candidate.Commands {
		line := strings.TrimSpace(commandLine)
		if line == "" {
			continue
		}

		cmd := system.NewShellCommand(line)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Run(); err == nil {
			result.Success = true
			result.Message = "installed"
			return result
		} else {
			lastErr = err.Error()
		}
	}

	if strings.TrimSpace(lastErr) == "" {
		lastErr = "installation command failed"
	}
	result.Message = fmt.Sprintf("failed: %s", lastErr)
	return result
}
