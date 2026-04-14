package cmd

import (
	"fmt"
	"m2apps/internal/config"

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
	},
}
