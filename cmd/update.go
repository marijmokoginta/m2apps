package cmd

import (
	"fmt"
	"m2apps/internal/ui"
	"m2apps/internal/updater"
	"os"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update <app_id>",
	Short: "Update installed application",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		appID := args[0]
		if err := updater.Update(appID); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}
	},
}
