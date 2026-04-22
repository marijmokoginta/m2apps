package fileops

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// RenameWithRetry renames src -> dst and retries only for Windows file-lock errors.
// On non-Windows platforms this is equivalent to os.Rename.
func RenameWithRetry(from, to string) error {
	src := filepath.Clean(strings.TrimSpace(from))
	dst := filepath.Clean(strings.TrimSpace(to))
	if src == "" || dst == "" {
		return fmt.Errorf("invalid rename path")
	}

	if runtime.GOOS != "windows" {
		return os.Rename(src, dst)
	}

	// Windows may keep handles for a short time after process termination or due to AV scans.
	const attempts = 7
	backoff := 200 * time.Millisecond
	var lastErr error
	for i := 0; i < attempts; i++ {
		err := os.Rename(src, dst)
		if err == nil {
			return nil
		}
		lastErr = err

		if !IsWindowsDirBusyError(err) {
			return err
		}

		time.Sleep(backoff)
		if backoff < 2*time.Second {
			backoff *= 2
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("unknown windows file lock")
	}
	return fmt.Errorf("rename failed after retries (source=%s, target=%s): %w", src, dst, lastErr)
}

// IsWindowsDirBusyError returns true when the error looks like a Windows
// sharing-violation / file-lock scenario.
func IsWindowsDirBusyError(err error) bool {
	if err == nil || runtime.GOOS != "windows" {
		return false
	}

	text := strings.ToLower(strings.TrimSpace(err.Error()))
	if matchesWindowsLockText(text) {
		return true
	}

	for unwrapped := errors.Unwrap(err); unwrapped != nil; unwrapped = errors.Unwrap(unwrapped) {
		if matchesWindowsLockText(strings.ToLower(strings.TrimSpace(unwrapped.Error()))) {
			return true
		}
	}

	return false
}

func matchesWindowsLockText(text string) bool {
	if strings.TrimSpace(text) == "" {
		return false
	}

	return strings.Contains(text, "being used by another process") ||
		strings.Contains(text, "used by another process") ||
		strings.Contains(text, "access is denied") ||
		strings.Contains(text, "permission denied") ||
		strings.Contains(text, "sharing violation")
}
