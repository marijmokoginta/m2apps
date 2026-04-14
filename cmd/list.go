package cmd

import (
	"fmt"
	"m2apps/internal/ui"
	"os"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed applications",
	Run: func(cmd *cobra.Command, args []string) {
		apps, err := loadInstalledApps()
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}

		if len(apps) == 0 {
			fmt.Println(ui.Warning("[WARN] No installed applications found."))
			return
		}

		fmt.Printf("%s %d\n", ui.Info("[INFO] Installed applications:"), len(apps))
		for _, app := range apps {
			if app.Name != "" && app.Name != app.ID {
				fmt.Printf("- %s (%s)\n", app.Name, app.ID)
				continue
			}
			fmt.Printf("- %s\n", app.ID)
		}
	},
}
