package daemon

import (
	"fmt"
	"m2apps/internal/system"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func daemonLogPath() string {
	return filepath.Join(system.GetLogDir(), "daemon.log")
}

func AppendRuntimeLog(level, message string) error {
	if strings.TrimSpace(message) == "" {
		return nil
	}

	if err := os.MkdirAll(system.GetLogDir(), 0o755); err != nil {
		return fmt.Errorf("failed to create daemon log directory: %w", err)
	}

	file, err := os.OpenFile(daemonLogPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open daemon log file: %w", err)
	}
	defer file.Close()

	tag := strings.ToUpper(strings.TrimSpace(level))
	if tag == "" {
		tag = "INFO"
	}

	_, err = fmt.Fprintf(
		file,
		"%s [%s] %s\n",
		time.Now().Format(time.RFC3339),
		tag,
		strings.TrimSpace(message),
	)
	if err != nil {
		return fmt.Errorf("failed to write daemon log file: %w", err)
	}
	return nil
}
