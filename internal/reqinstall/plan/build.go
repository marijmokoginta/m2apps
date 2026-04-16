package plan

import (
	"fmt"
	"m2apps/internal/reqinstall/resolver"
	"m2apps/internal/requirements"
	"runtime"
	"strings"
)

func Build(reqs []requirements.Requirement, results []requirements.Result) InstallPlan {
	osName := runtime.GOOS
	out := InstallPlan{
		Missing:    make([]MissingRequirement, 0),
		Candidates: make([]InstallCandidate, 0),
		Warnings:   make([]string, 0),
	}

	for i, res := range results {
		if res.Success {
			continue
		}

		toolType := resolveToolType(reqs, i, res)
		missing := MissingRequirement{
			ToolType: toolType,
			Name:     resolveToolName(toolType, res.Name),
			Required: strings.TrimSpace(res.Required),
			Found:    strings.TrimSpace(res.Found),
			Reason:   strings.TrimSpace(res.Reason),
		}
		out.Missing = append(out.Missing, missing)

		candidate, ok := buildCandidate(osName, missing)
		if !ok {
			out.Warnings = append(out.Warnings, fmt.Sprintf("No installer strategy for %s on %s", missing.Name, osName))
			continue
		}
		out.Candidates = append(out.Candidates, candidate)
	}

	return out
}

func resolveToolType(reqs []requirements.Requirement, idx int, res requirements.Result) string {
	if idx >= 0 && idx < len(reqs) {
		if t := strings.ToLower(strings.TrimSpace(reqs[idx].Type)); t != "" {
			return t
		}
	}

	name := strings.ToLower(strings.TrimSpace(res.Name))
	switch name {
	case "php", "node", "mysql", "flutter", "dart":
		return name
	default:
		return name
	}
}

func resolveToolName(toolType, fallback string) string {
	switch strings.TrimSpace(toolType) {
	case "php":
		return "PHP"
	case "node":
		return "Node"
	case "mysql":
		return "MySQL"
	case "flutter":
		return "Flutter"
	case "dart":
		return "Dart"
	default:
		if strings.TrimSpace(fallback) != "" {
			return fallback
		}
		return strings.ToUpper(toolType)
	}
}

func buildCandidate(osName string, missing MissingRequirement) (InstallCandidate, bool) {
	target := resolver.Resolve(missing.ToolType, missing.Required)
	if target == "" {
		target = "stable"
	}

	candidate := InstallCandidate{
		ToolType:        missing.ToolType,
		Name:            missing.Name,
		RequiredVersion: missing.Required,
		TargetVersion:   target,
		OS:              osName,
	}

	switch osName {
	case "linux":
		candidate.Method = "package-manager"
		candidate.Commands = buildLinuxCommands(missing.ToolType, target)
		candidate.Notes = "Requires sudo/root and active package manager"
	case "darwin":
		candidate.Method = "homebrew"
		candidate.Commands = buildDarwinCommands(missing.ToolType, target)
		candidate.Notes = "Requires Homebrew"
	case "windows":
		candidate.Method = "winget/choco"
		candidate.Commands = buildWindowsCommands(missing.ToolType, target)
		candidate.Notes = "Requires Administrator privileges"
	default:
		return InstallCandidate{}, false
	}

	if len(candidate.Commands) == 0 {
		return InstallCandidate{}, false
	}

	return candidate, true
}

func buildLinuxCommands(tool string, _ string) []string {
	switch tool {
	case "php":
		return []string{
			"if command -v apt-get >/dev/null 2>&1; then sudo apt-get update && sudo apt-get install -y php-cli; elif command -v dnf >/dev/null 2>&1; then sudo dnf install -y php-cli; elif command -v yum >/dev/null 2>&1; then sudo yum install -y php-cli; else exit 127; fi",
		}
	case "node":
		return []string{
			"if command -v apt-get >/dev/null 2>&1; then sudo apt-get update && sudo apt-get install -y nodejs npm; elif command -v dnf >/dev/null 2>&1; then sudo dnf install -y nodejs npm; elif command -v yum >/dev/null 2>&1; then sudo yum install -y nodejs npm; else exit 127; fi",
		}
	case "mysql":
		return []string{
			"if command -v apt-get >/dev/null 2>&1; then sudo apt-get update && sudo apt-get install -y mysql-client; elif command -v dnf >/dev/null 2>&1; then sudo dnf install -y mysql; elif command -v yum >/dev/null 2>&1; then sudo yum install -y mysql; else exit 127; fi",
		}
	case "flutter":
		return []string{
			"if command -v snap >/dev/null 2>&1; then sudo snap install flutter --classic; else exit 127; fi",
		}
	case "dart":
		return []string{
			"if command -v apt-get >/dev/null 2>&1; then sudo apt-get update && sudo apt-get install -y dart; elif command -v snap >/dev/null 2>&1; then sudo snap install dart --classic; else exit 127; fi",
		}
	default:
		return nil
	}
}

func buildDarwinCommands(tool string, _ string) []string {
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
		return []string{"brew tap dart-lang/dart && brew install dart"}
	default:
		return nil
	}
}

func buildWindowsCommands(tool string, targetVersion string) []string {
	versionArg := ""
	if strings.TrimSpace(targetVersion) != "" && strings.TrimSpace(targetVersion) != "stable" {
		versionArg = "--version " + targetVersion + " "
	}

	switch tool {
	case "php":
		return []string{
			"where winget >nul 2>nul && winget install -e --id PHP.PHP " + versionArg + "|| (where choco >nul 2>nul && choco install php -y)",
		}
	case "node":
		return []string{
			"where winget >nul 2>nul && winget install -e --id OpenJS.NodeJS " + versionArg + "|| (where choco >nul 2>nul && choco install nodejs-lts -y)",
		}
	case "mysql":
		return []string{
			"where winget >nul 2>nul && winget install -e --id Oracle.MySQL " + versionArg + "|| (where choco >nul 2>nul && choco install mysql -y)",
		}
	case "flutter":
		return []string{
			"where winget >nul 2>nul && winget install -e --id Google.Flutter " + versionArg + "|| (where choco >nul 2>nul && choco install flutter -y)",
		}
	case "dart":
		return []string{
			"where winget >nul 2>nul && winget install -e --id Dart.DartSDK " + versionArg + "|| (where choco >nul 2>nul && choco install dart-sdk -y)",
		}
	default:
		return nil
	}
}
