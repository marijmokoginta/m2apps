package reqinstall

import (
	"fmt"
	"m2apps/internal/config"
	"m2apps/internal/reqinstall/executor"
	"m2apps/internal/reqinstall/manual"
	"m2apps/internal/reqinstall/plan"
	"m2apps/internal/requirements"
	"m2apps/internal/ui"
	"runtime"
	"strings"
)

func ResolveAndInstall(mode string, reqs []requirements.Requirement, initial []requirements.Result) ([]requirements.Result, error) {
	installMode := normalizeMode(mode)
	if !hasMissing(initial) {
		return initial, nil
	}

	installPlan := plan.Build(reqs, initial)
	printMissing(installPlan)

	if installMode == config.InstallModeManual {
		printManual(installPlan)
		return initial, fmt.Errorf("missing requirements detected (manual mode)")
	}

	if !isInteractiveTerminal() {
		fmt.Println(ui.Warning("[WARN] Assisted requirement installation requires interactive terminal."))
		printManual(installPlan)
		return initial, fmt.Errorf("missing requirements detected")
	}

	printPlan(installPlan)

	if len(installPlan.Candidates) == 0 {
		printManual(installPlan)
		return initial, fmt.Errorf("no installer strategy available for missing requirements")
	}

	if !confirm("Proceed with assisted requirement installation?") {
		fmt.Println(ui.Warning("[WARN] Requirement installation cancelled by user."))
		printManual(installPlan)
		return initial, fmt.Errorf("missing requirements detected")
	}

	exec := executor.New()
	fmt.Println()
	fmt.Println(ui.Info("[INFO] Executing install plan..."))
	for _, candidate := range installPlan.Candidates {
		if !confirm(fmt.Sprintf("Install %s (%s) using %s?", candidate.Name, candidate.TargetVersion, candidate.Method)) {
			fmt.Println(ui.Warning(fmt.Sprintf("[SKIP] %s", candidate.Name)))
			continue
		}

		result := exec.Execute(candidate)
		if result.Success {
			fmt.Println(ui.Success(fmt.Sprintf("[OK] %s installed", candidate.Name)))
			if runtime.GOOS == "windows" {
				fmt.Println(ui.Info("[INFO] Windows detected: if installer updated PATH/env, restart terminal so new commands are available."))
			}
		} else {
			fmt.Println(ui.Error(fmt.Sprintf("[FAIL] %s (%s)", candidate.Name, result.Message)))
		}
	}

	fmt.Println()
	fmt.Println(ui.Info("[INFO] Re-checking requirements..."))
	after := requirements.Run(reqs)
	if hasMissing(after) {
		remaining := plan.Build(reqs, after)
		fmt.Println(ui.Error("[ERROR] Some requirements are still missing."))
		printManual(remaining)
		return after, fmt.Errorf("requirements are not satisfied after assisted installation")
	}

	fmt.Println(ui.Success("[OK] All requirements are satisfied."))
	return after, nil
}

func normalizeMode(mode string) string {
	raw := strings.ToLower(strings.TrimSpace(mode))
	if raw == "" {
		return config.InstallModeAssisted
	}
	if raw == config.InstallModeManual {
		return config.InstallModeManual
	}
	return config.InstallModeAssisted
}

func hasMissing(results []requirements.Result) bool {
	for _, res := range results {
		if !res.Success {
			return true
		}
	}
	return false
}

func printMissing(installPlan plan.InstallPlan) {
	if len(installPlan.Missing) == 0 {
		return
	}

	fmt.Println()
	fmt.Println(ui.Warning("[WARN] Missing tools detected:"))
	for _, item := range installPlan.Missing {
		found := strings.TrimSpace(item.Found)
		if found == "" {
			found = "not available"
		}
		reason := strings.TrimSpace(item.Reason)
		if reason == "" {
			reason = "unknown"
		}
		fmt.Printf("- [X] %s %s (found: %s, reason: %s)\n", item.Name, item.Required, found, reason)
	}
}

func printPlan(installPlan plan.InstallPlan) {
	fmt.Println()
	fmt.Println(ui.Info("[INFO] Install plan:"))
	for i, c := range installPlan.Candidates {
		fmt.Printf("%d. %s -> target %s via %s\n", i+1, c.Name, c.TargetVersion, c.Method)
		if strings.TrimSpace(c.Notes) != "" {
			fmt.Printf("   note: %s\n", c.Notes)
		}
	}

	for _, warning := range installPlan.Warnings {
		fmt.Println(ui.Warning(fmt.Sprintf("[WARN] %s", warning)))
	}
}

func printManual(installPlan plan.InstallPlan) {
	text := manual.RenderInstructions(installPlan.Missing)
	if strings.TrimSpace(text) == "" {
		return
	}
	fmt.Println()
	fmt.Println(ui.Info("[INFO] Manual fallback instructions:"))
	fmt.Println(text)
}
