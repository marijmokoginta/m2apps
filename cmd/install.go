package cmd

import (
	"fmt"
	"m2apps/internal/config"
	"m2apps/internal/requirements"
	_ "m2apps/internal/requirements/checkers"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install application from install.json",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Reading install.json...")

		cfg, err := config.LoadFromFile("install.json")
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		if err := cfg.Validate(); err != nil {
			fmt.Println("Error in install.json:")
			fmt.Println(err)
			return
		}

		fmt.Println("Config loaded")
		fmt.Printf("App: %s\n", cfg.Name)
		fmt.Printf("Preset: %s\n", cfg.Preset)

		reqs := make([]requirements.Requirement, 0, len(cfg.Requirements))
		for _, req := range cfg.Requirements {
			reqs = append(reqs, requirements.Requirement{
				Type:    req.Type,
				Version: req.Version,
			})
		}

		results := requirements.Run(reqs)
		hasFailure := printRequirementResults(results)
		if hasFailure {
			fmt.Println()
			fmt.Println("Installation aborted.")
			os.Exit(1)
		}
	},
}

func printRequirementResults(results []requirements.Result) bool {
	fmt.Println()
	fmt.Println("Checking requirements...")
	fmt.Println()

	hasFailure := false

	for _, res := range results {
		label := formatRequirementLabel(res)

		if res.Success {
			fmt.Printf("[✓] %s (found %s)\n", label, res.Found)
			continue
		}

		hasFailure = true

		switch {
		case res.Found == "not found":
			fmt.Printf("[✗] %s (not found)\n", label)
		case strings.TrimSpace(res.Found) != "":
			fmt.Printf("[✗] %s (found %s)\n", label, res.Found)
		case strings.TrimSpace(res.Message) != "":
			fmt.Printf("[✗] %s (%s)\n", label, res.Message)
		default:
			fmt.Printf("[✗] %s (failed)\n", label)
		}
	}

	return hasFailure
}

func formatRequirementLabel(res requirements.Result) string {
	if strings.TrimSpace(res.Required) == "" {
		return res.Name
	}
	return fmt.Sprintf("%s %s", res.Name, res.Required)
}
