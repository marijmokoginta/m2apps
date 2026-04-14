package system

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func NewCommand(cmd string, args ...string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", append([]string{"/C", cmd}, args...)...)
	}
	return exec.Command(cmd, args...)
}

func NewProcessCommand(cmd string, args ...string) *exec.Cmd {
	return exec.Command(cmd, args...)
}

func NewShellCommand(command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/C", command)
	}
	return exec.Command("sh", "-c", command)
}

func RunCommand(cmd string, args ...string) error {
	command := NewCommand(cmd, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

func CombinedOutput(cmd string, args ...string) ([]byte, error) {
	command := NewCommand(cmd, args...)
	return command.CombinedOutput()
}

func IsCommandNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, exec.ErrNotFound) {
		return true
	}

	text := strings.ToLower(err.Error())
	return strings.Contains(text, "executable file not found") ||
		strings.Contains(text, "not recognized as an internal or external command") ||
		strings.Contains(text, "cannot find the file")
}
