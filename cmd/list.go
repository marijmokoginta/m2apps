package cmd

import (
	"fmt"
	"m2apps/internal/process"
	"m2apps/internal/ui"
	"os"
	"strings"

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
		fmt.Println(renderAppsTable(apps, processManager))
	},
}

type appRunStatus struct {
	Base  string
	Label string
}

func renderAppsTable(apps []installedApp, processManager *process.Manager) string {
	headers := []string{"APP ID", "NAME", "VERSION", "CHANNEL", "PRESET", "STATUS", "INSTALL PATH"}
	statusIndex := 5

	rows := make([][]string, 0, len(apps))
	statusBases := make([]string, 0, len(apps))

	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}

	for _, app := range apps {
		status := resolveAppRunningStatus(processManager, app.ID)
		row := []string{
			coalesce(app.ID),
			coalesce(app.Name),
			coalesce(app.Version),
			coalesce(app.Channel),
			coalesce(app.Preset),
			coalesce(status.Label),
			coalesce(app.InstallPath),
		}
		rows = append(rows, row)
		statusBases = append(statusBases, status.Base)

		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	border := func() string {
		parts := make([]string, 0, len(widths))
		for _, width := range widths {
			parts = append(parts, strings.Repeat("-", width+2))
		}
		return "+" + strings.Join(parts, "+") + "+"
	}

	formatRow := func(cols []string, statusBase string) string {
		cells := make([]string, 0, len(cols))
		for i, col := range cols {
			padded := fmt.Sprintf("%-*s", widths[i], col)
			if i == statusIndex {
				raw := statusBase
				if strings.TrimSpace(raw) == "" {
					raw = col
				}
				padded = colorizeProcessStatusCell(raw, padded)
			}
			cells = append(cells, " "+padded+" ")
		}
		return "|" + strings.Join(cells, "|") + "|"
	}

	lines := []string{
		border(),
		formatRow(headers, ""),
		border(),
	}

	for i, row := range rows {
		lines = append(lines, formatRow(row, statusBases[i]))
	}
	lines = append(lines, border())

	return strings.Join(lines, "\n")
}

func resolveAppRunningStatus(processManager *process.Manager, appID string) appRunStatus {
	status, err := processManager.Status(appID)
	if err != nil {
		return appRunStatus{
			Base:  "unknown",
			Label: "unknown",
		}
	}

	if len(status.Processes) == 0 {
		return appRunStatus{
			Base:  "stopped",
			Label: "stopped",
		}
	}

	running := 0
	for _, proc := range status.Processes {
		if strings.EqualFold(strings.TrimSpace(proc.Status), "running") {
			running++
		}
	}

	if running == 0 {
		return appRunStatus{
			Base:  "stopped",
			Label: fmt.Sprintf("stopped (0/%d)", len(status.Processes)),
		}
	}

	label := fmt.Sprintf("running (%d/%d)", running, len(status.Processes))
	return appRunStatus{
		Base:  "running",
		Label: label,
	}
}

func coalesce(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "-"
	}
	return trimmed
}
