package config

import (
	"fmt"
	"m2apps/internal/hostmode"
	"strings"
)

const (
	InstallModeAssisted = "assisted"
	InstallModeManual   = "manual"
)

func (c *InstallConfig) Validate() error {
	var errors []string

	if strings.TrimSpace(c.AppID) == "" {
		errors = append(errors, "app_id is required")
	}

	if strings.TrimSpace(c.Name) == "" {
		errors = append(errors, "name is required")
	}

	if strings.TrimSpace(c.Source.Type) == "" {
		errors = append(errors, "source.type is required")
	}

	if strings.TrimSpace(c.Source.Repo) == "" {
		errors = append(errors, "source.repo is required")
	}

	if strings.TrimSpace(c.Source.Version) == "" {
		errors = append(errors, "source.version is required")
	}

	if strings.TrimSpace(c.Source.Asset) == "" {
		errors = append(errors, "source.asset is required")
	}

	if strings.TrimSpace(c.Auth.Type) == "" {
		errors = append(errors, "auth.type is required")
	}

	if strings.TrimSpace(c.Auth.Value) == "" {
		errors = append(errors, "auth.value is required")
	}

	if strings.TrimSpace(c.Preset) == "" {
		errors = append(errors, "preset is required")
	}

	installMode := strings.ToLower(strings.TrimSpace(c.InstallMode))
	if installMode == "" {
		installMode = InstallModeAssisted
	}
	if installMode != InstallModeAssisted && installMode != InstallModeManual {
		errors = append(errors, "install_mode must be one of: assisted, manual")
	}
	c.InstallMode = installMode

	if !hostmode.IsValid(c.ServerMode) {
		errors = append(errors, "server_mode must be one of: localhost, lan")
	}

	if hostmode.Normalize(c.ServerMode) == hostmode.LAN && !isLaravelPreset(c.Preset) {
		errors = append(errors, "server_mode=lan is only supported for preset: laravel, laravel-inertia")
	}

	channel := strings.ToLower(strings.TrimSpace(c.Channel))
	if channel != "" && channel != "stable" && channel != "beta" && channel != "alpha" {
		errors = append(errors, "channel must be one of: stable, beta, alpha")
	}

	if len(c.Requirements) == 0 {
		errors = append(errors, "requirements is required")
	}

	for i, req := range c.Requirements {
		if strings.TrimSpace(req.Type) == "" {
			errors = append(errors, fmt.Sprintf("requirements[%d].type is required", i))
		}

		if strings.TrimSpace(req.Version) == "" {
			errors = append(errors, fmt.Sprintf("requirements[%d].version is required", i))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("config validation failed:\n- %s", joinErrors(errors))
	}

	return nil
}

func isLaravelPreset(preset string) bool {
	switch strings.ToLower(strings.TrimSpace(preset)) {
	case "laravel", "laravel-inertia":
		return true
	default:
		return false
	}
}

func joinErrors(errs []string) string {
	result := ""
	for i, e := range errs {
		if i == 0 {
			result += e
		} else {
			result += "\n- " + e
		}
	}
	return result
}
