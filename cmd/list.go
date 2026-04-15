package cmd

import (
	"fmt"
	"m2apps/internal/process"
	"m2apps/internal/ui"
	"os"
	"strings"
	"text/tabwriter"

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

		processManager, err := process.NewManager()
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}

		fmt.Printf("%s %d\n\n", ui.Info("[INFO] Installed applications:"), len(apps))
		renderAppsTable(apps, processManager)
	},
}

func renderAppsTable(apps []installedApp, processManager *process.Manager) {
	writer := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "APP ID\tNAME\tVERSION\tCHANNEL\tPRESET\tSTATUS\tINSTALL PATH")

	for _, app := range apps {
		status := resolveAppRunningStatus(processManager, app.ID)
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			coalesce(app.ID),
			coalesce(app.Name),
			coalesce(app.Version),
			coalesce(app.Channel),
			coalesce(app.Preset),
			status,
			coalesce(app.InstallPath),
		)
	}

	_ = writer.Flush()
}

func resolveAppRunningStatus(processManager *process.Manager, appID string) string {
	status, err := processManager.Status(appID)
	if err != nil {
		return "unknown"
	}

	if len(status.Processes) == 0 {
		return "stopped"
	}

	running := 0
	for _, proc := range status.Processes {
		if strings.EqualFold(strings.TrimSpace(proc.Status), "running") {
			running++
		}
	}

	if running == 0 {
		return "stopped"
	}

	return fmt.Sprintf("running (%d/%d)", running, len(status.Processes))
}

func coalesce(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "-"
	}
	return trimmed
}
