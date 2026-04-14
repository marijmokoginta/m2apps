package checkers

import (
	"fmt"
	"m2apps/internal/system"
	"strings"

	"m2apps/internal/requirements"
)

func checkTool(displayName, command string, args []string, constraint string) (requirements.Result, error) {
	res := requirements.Result{
		Name:     displayName,
		Required: constraint,
	}

	output, err := runVersionCommand(command, args...)
	if err != nil {
		if err.Error() == "not found" {
			res.Found = "not found"
		}
		return res, err
	}

	version, err := requirements.ParseVersion(output)
	if err != nil {
		return res, fmt.Errorf("invalid version output")
	}

	res.Found = version.String()

	ok, err := requirements.Satisfies(version, constraint)
	if err != nil {
		return res, err
	}

	res.Success = ok
	return res, nil
}

func runVersionCommand(name string, args ...string) (string, error) {
	output, err := system.CombinedOutput(name, args...)
	if err != nil {
		if system.IsCommandNotFound(err) {
			return "", fmt.Errorf("not found")
		}

		trimmed := strings.TrimSpace(string(output))
		if trimmed != "" {
			return trimmed, nil
		}

		return "", fmt.Errorf("failed to run %s", name)
	}

	return strings.TrimSpace(string(output)), nil
}
