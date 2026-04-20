package preset

import (
	"m2apps/internal/dbsetup"
	"os"
	"path/filepath"
	"strings"
)

// dbRequiredPresets lists presets that require database configuration.
var dbRequiredPresets = map[string]bool{
	"laravel":         true,
	"laravel-inertia": true,
}

// RequiresDBSetup reports whether the given preset requires database configuration.
func RequiresDBSetup(preset string) bool {
	return dbRequiredPresets[normalizePreset(preset)]
}

func IsLaravelPreset(preset string) bool {
	switch normalizePreset(preset) {
	case "laravel", "laravel-inertia":
		return true
	default:
		return false
	}
}

// ReadDBDefaults reads the default database configuration for the given preset
// by parsing .env.example (or similar reference files) in workDir.
// Returns a DBConfig populated with whatever defaults can be found; missing
// values fall back to sensible built-in defaults.
func ReadDBDefaults(preset, workDir string) dbsetup.DBConfig {
	normalized := normalizePreset(preset)

	switch normalized {
	case "laravel", "laravel-inertia":
		return readLaravelDBDefaults(workDir)
	default:
		return dbsetup.DBConfig{
			Driver: "mysql",
			Host:   "127.0.0.1",
			Port:   "3306",
		}
	}
}

// readLaravelDBDefaults parses .env.example in the Laravel project directory.
func readLaravelDBDefaults(workDir string) dbsetup.DBConfig {
	defaults := dbsetup.DBConfig{
		Driver: "mysql",
		Host:   "127.0.0.1",
		Port:   "3306",
	}

	examplePath := filepath.Join(workDir, ".env.example")
	content, err := os.ReadFile(examplePath)
	if err != nil {
		// .env.example not found or unreadable — return built-in defaults
		return defaults
	}

	vals := parseEnvFile(content)

	if v := vals["DB_CONNECTION"]; v != "" {
		defaults.Driver = v
	}
	if v := vals["DB_HOST"]; v != "" {
		defaults.Host = v
	}
	if v := vals["DB_PORT"]; v != "" {
		defaults.Port = v
	}
	if v := vals["DB_DATABASE"]; v != "" {
		defaults.DBName = v
	}
	if v := vals["DB_USERNAME"]; v != "" {
		defaults.Username = v
	}
	if v := vals["DB_PASSWORD"]; v != "" {
		defaults.Password = v
	}

	return defaults
}

// parseEnvFile parses a .env / .env.example byte slice into a key→value map.
// Comments and blank lines are ignored; values are returned as-is (unquoted).
func parseEnvFile(content []byte) map[string]string {
	result := map[string]string{}
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
		val := strings.TrimSpace(parts[1])
		// Strip surrounding quotes if present
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		if key != "" {
			result[key] = val
		}
	}
	return result
}
