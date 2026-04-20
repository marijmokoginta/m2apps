package env

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func Inject(installPath string, vars map[string]string) error {
	return injectWithPlaceholders(installPath, vars, nil)
}

func Upsert(installPath string, vars map[string]string) error {
	_, err := UpsertWithResult(installPath, vars)
	return err
}

func DeleteKeys(installPath string, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	targetFile, err := resolveExistingEnvFile(installPath)
	if err != nil {
		return err
	}
	if strings.TrimSpace(targetFile) == "" {
		return nil
	}

	content, err := os.ReadFile(targetFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read env file %s: %w", targetFile, err)
	}

	keySet := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		keySet[key] = struct{}{}
	}
	if len(keySet) == 0 {
		return nil
	}

	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			filtered = append(filtered, line)
			continue
		}

		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) == 0 {
			filtered = append(filtered, line)
			continue
		}

		key := strings.TrimSpace(parts[0])
		if key == "" {
			filtered = append(filtered, line)
			continue
		}

		if _, shouldDelete := keySet[key]; shouldDelete {
			continue
		}
		filtered = append(filtered, line)
	}

	updated := strings.Join(filtered, "\n")
	updated = strings.TrimRight(updated, "\n") + "\n"
	if err := os.WriteFile(targetFile, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("failed to write env file %s: %w", targetFile, err)
	}
	return nil
}

func UpsertWithResult(installPath string, vars map[string]string) (bool, error) {
	targetFile, err := resolveTargetEnvFile(installPath)
	if err != nil {
		return false, err
	}

	content, err := os.ReadFile(targetFile)
	if err != nil {
		return false, fmt.Errorf("failed to read env file %s: %w", targetFile, err)
	}

	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	replaced := make(map[string]struct{}, len(vars))
	changed := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		parts := strings.SplitN(trimmed, "=", 2)
		key := strings.TrimSpace(parts[0])
		value, ok := vars[key]
		if !ok {
			continue
		}

		if len(parts) == 2 && strings.TrimSpace(parts[1]) == value {
			replaced[key] = struct{}{}
			continue
		}

		lines[i] = fmt.Sprintf("%s=%s", key, value)
		replaced[key] = struct{}{}
		changed = true
	}

	for key, value := range vars {
		if _, ok := replaced[key]; ok {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
		changed = true
	}

	updated := strings.Join(lines, "\n")
	updated = strings.TrimRight(updated, "\n") + "\n"
	if err := os.WriteFile(targetFile, []byte(updated), 0o644); err != nil {
		return false, fmt.Errorf("failed to write env file %s: %w", targetFile, err)
	}

	return changed, nil
}

func InjectAppURL(installPath string, port int) error {
	if port <= 0 {
		return nil
	}

	return injectWithPlaceholders(
		installPath,
		map[string]string{
			"APP_URL": "http://127.0.0.1:{PORT}",
		},
		map[string]string{
			"PORT": strconv.Itoa(port),
		},
	)
}

func injectWithPlaceholders(installPath string, vars map[string]string, placeholders map[string]string) error {
	targetFile, err := resolveTargetEnvFile(installPath)
	if err != nil {
		return err
	}

	entries, err := readEnvKeys(targetFile)
	if err != nil {
		return err
	}

	var lines []string
	for key, value := range vars {
		if _, exists := entries[key]; exists {
			continue
		}

		resolved := applyPlaceholders(value, placeholders)
		lines = append(lines, fmt.Sprintf("%s=%s", key, resolved))
	}

	if len(lines) == 0 {
		return nil
	}

	file, err := os.OpenFile(targetFile, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open env file %s: %w", targetFile, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat env file %s: %w", targetFile, err)
	}
	if info.Size() > 0 {
		if _, err := file.WriteString("\n"); err != nil {
			return fmt.Errorf("failed to write env separator: %w", err)
		}
	}

	for _, line := range lines {
		if _, err := file.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("failed to inject env variable: %w", err)
		}
	}

	return nil
}

func applyPlaceholders(value string, placeholders map[string]string) string {
	if strings.TrimSpace(value) == "" || len(placeholders) == 0 {
		return value
	}

	resolved := value
	for key, replacement := range placeholders {
		token := "{" + strings.TrimSpace(key) + "}"
		resolved = strings.ReplaceAll(resolved, token, replacement)
	}
	return resolved
}

func resolveTargetEnvFile(installPath string) (string, error) {
	candidates := []string{
		filepath.Join(installPath, ".env"),
		filepath.Join(installPath, ".env.local"),
	}

	for _, file := range candidates {
		if _, err := os.Stat(file); err == nil {
			return file, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to check env file %s: %w", file, err)
		}
	}

	target := filepath.Join(installPath, ".env")
	examplePath := filepath.Join(installPath, ".env.example")
	if content, err := os.ReadFile(examplePath); err == nil {
		if err := os.WriteFile(target, content, 0o644); err != nil {
			return "", fmt.Errorf("failed to create env file %s from .env.example: %w", target, err)
		}
		return target, nil
	} else if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read env example file %s: %w", examplePath, err)
	}

	if err := os.WriteFile(target, []byte(""), 0o644); err != nil {
		return "", fmt.Errorf("failed to create env file %s: %w", target, err)
	}
	return target, nil
}

func resolveExistingEnvFile(installPath string) (string, error) {
	candidates := []string{
		filepath.Join(installPath, ".env"),
		filepath.Join(installPath, ".env.local"),
	}

	for _, file := range candidates {
		if _, err := os.Stat(file); err == nil {
			return file, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to check env file %s: %w", file, err)
		}
	}

	return "", nil
}

func readEnvKeys(path string) (map[string]struct{}, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read env file %s: %w", path, err)
	}

	keys := make(map[string]struct{})
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		key := strings.TrimSpace(parts[0])
		if key != "" {
			keys[key] = struct{}{}
		}
	}
	return keys, nil
}

// ReadValues reads specific key values from the .env file in the given install path.
// Keys not found in the .env are returned as empty strings (not an error).
func ReadValues(installPath string, keys []string) (map[string]string, error) {
	if len(keys) == 0 {
		return map[string]string{}, nil
	}

	targetFile, err := resolveExistingEnvFile(installPath)
	if err != nil {
		return nil, err
	}
	if targetFile == "" {
		result := make(map[string]string, len(keys))
		for _, k := range keys {
			result[strings.TrimSpace(k)] = ""
		}
		return result, nil
	}

	content, err := os.ReadFile(targetFile)
	if err != nil {
		if os.IsNotExist(err) {
			result := make(map[string]string, len(keys))
			for _, k := range keys {
				result[strings.TrimSpace(k)] = ""
			}
			return result, nil
		}
		return nil, fmt.Errorf("failed to read env file %s: %w", targetFile, err)
	}

	keySet := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k != "" {
			keySet[k] = struct{}{}
		}
	}

	result := make(map[string]string, len(keys))
	for _, line := range strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		if _, wanted := keySet[key]; !wanted {
			continue
		}
		result[key] = strings.TrimSpace(parts[1])
	}

	// Fill missing keys with empty string
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if _, found := result[k]; !found {
			result[k] = ""
		}
	}

	return result, nil
}
