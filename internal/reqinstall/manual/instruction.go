package manual

import (
	"fmt"
	"m2apps/internal/reqinstall/plan"
	"runtime"
	"strings"
)

func RenderInstructions(missing []plan.MissingRequirement) string {
	if len(missing) == 0 {
		return ""
	}

	osName := runtime.GOOS
	lines := []string{"Manual installation steps:"}
	for _, item := range missing {
		tool := strings.ToLower(strings.TrimSpace(item.ToolType))
		if tool == "" {
			tool = strings.ToLower(strings.TrimSpace(item.Name))
		}
		lines = append(lines, fmt.Sprintf("- %s (%s):", item.Name, item.Required))
		for _, cmd := range commandsFor(osName, tool) {
			lines = append(lines, fmt.Sprintf("  %s", cmd))
		}
	}

	return strings.Join(lines, "\n")
}

func commandsFor(osName, tool string) []string {
	switch osName {
	case "linux":
		switch tool {
		case "php":
			return []string{"sudo apt-get install -y php-cli (or dnf/yum equivalent)"}
		case "node":
			return []string{"sudo apt-get install -y nodejs npm (or dnf/yum equivalent)"}
		case "mysql":
			return []string{"sudo apt-get install -y mysql-client (or dnf/yum equivalent)"}
		case "flutter":
			return []string{"sudo snap install flutter --classic"}
		case "dart":
			return []string{"sudo apt-get install -y dart (or snap install dart --classic)"}
		}
	case "darwin":
		switch tool {
		case "php":
			return []string{"brew install php"}
		case "node":
			return []string{"brew install node"}
		case "mysql":
			return []string{"brew install mysql-client"}
		case "flutter":
			return []string{"brew install --cask flutter"}
		case "dart":
			return []string{"brew tap dart-lang/dart", "brew install dart"}
		}
	case "windows":
		switch tool {
		case "php":
			return []string{"winget install -e --id PHP.PHP"}
		case "node":
			return []string{"winget install -e --id OpenJS.NodeJS"}
		case "mysql":
			return []string{"winget install -e --id Oracle.MySQL"}
		case "flutter":
			return []string{"winget install -e --id Google.Flutter"}
		case "dart":
			return []string{"winget install -e --id Dart.DartSDK"}
		}
	}

	return []string{"Please install from official vendor documentation and ensure the tool is available in PATH."}
}
