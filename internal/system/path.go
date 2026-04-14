package system

import (
	"os"
	"path/filepath"
	"strings"
)

func GetBaseDir() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ".m2apps"
	}
	return filepath.Join(home, ".m2apps")
}

func GetAppsDir() string {
	return filepath.Join(GetBaseDir(), "apps")
}

func GetAppDir(appID string) string {
	return filepath.Join(GetAppsDir(), strings.TrimSpace(appID))
}

func GetLogDir() string {
	return filepath.Join(GetBaseDir(), "logs")
}

func GetDaemonDir() string {
	return filepath.Join(GetBaseDir(), "daemon")
}
