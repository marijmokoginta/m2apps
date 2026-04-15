package cmd

import (
	"fmt"
	"m2apps/internal/process"
	"m2apps/internal/ui"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "Manage installed app processes",
}

var appStartCmd = &cobra.Command{
	Use:   "start <app_id>",
	Short: "Start app processes",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputAppID := strings.TrimSpace(args[0])
		appID, err := resolveInstalledAppID(inputAppID)
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}
		if appID != inputAppID {
			fmt.Println(ui.Info(fmt.Sprintf("[INFO] Using installed app_id: %s", appID)))
		}

		manager, err := process.NewManager()
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}

		fmt.Println(ui.Info(fmt.Sprintf("[INFO] Starting app %s...", appID)))
		state, err := manager.Start(appID)
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}

		fmt.Println(ui.Success(fmt.Sprintf("[OK] Started %d process(es) for %s", len(state.Processes), appID)))
	},
}

var appStopCmd = &cobra.Command{
	Use:   "stop <app_id>",
	Short: "Stop app processes",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputAppID := strings.TrimSpace(args[0])
		appID, err := resolveInstalledAppID(inputAppID)
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}
		if appID != inputAppID {
			fmt.Println(ui.Info(fmt.Sprintf("[INFO] Using installed app_id: %s", appID)))
		}

		manager, err := process.NewManager()
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}

		fmt.Println(ui.Info(fmt.Sprintf("[INFO] Stopping app %s...", appID)))
		state, err := manager.Stop(appID)
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}

		if len(state.Processes) == 0 {
			fmt.Println(ui.Warning(fmt.Sprintf("[WARN] No process records found for %s", appID)))
			return
		}

		fmt.Println(ui.Success(fmt.Sprintf("[OK] Stopped app %s", appID)))
	},
}

var appRestartCmd = &cobra.Command{
	Use:   "restart <app_id>",
	Short: "Restart app processes",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputAppID := strings.TrimSpace(args[0])
		appID, err := resolveInstalledAppID(inputAppID)
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}
		if appID != inputAppID {
			fmt.Println(ui.Info(fmt.Sprintf("[INFO] Using installed app_id: %s", appID)))
		}

		manager, err := process.NewManager()
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}

		fmt.Println(ui.Info(fmt.Sprintf("[INFO] Restarting app %s...", appID)))
		state, err := manager.Restart(appID)
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}

		fmt.Println(ui.Success(fmt.Sprintf("[OK] Restarted app %s (%d process(es))", appID, len(state.Processes))))
	},
}

var appStatusCmd = &cobra.Command{
	Use:   "status <app_id>",
	Short: "Show app process status",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputAppID := strings.TrimSpace(args[0])
		appID, err := resolveInstalledAppID(inputAppID)
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}
		if appID != inputAppID {
			fmt.Println(ui.Info(fmt.Sprintf("[INFO] Using installed app_id: %s", appID)))
		}

		manager, err := process.NewManager()
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}

		state, err := manager.Status(appID)
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}

		if len(state.Processes) == 0 {
			fmt.Println(ui.Warning(fmt.Sprintf("[WARN] No process records found for %s", appID)))
			return
		}

		fmt.Println(ui.Info(fmt.Sprintf("[INFO] App %s process status:", appID)))
		for _, proc := range state.Processes {
			command := strings.Join(proc.Command, " ")
			if command == "" {
				command = "-"
			}

			fmt.Printf("- %s | pid: %d | status: %s | cmd: %s\n", proc.Name, proc.PID, proc.Status, command)
		}
	},
}

func init() {
	appCmd.AddCommand(appStartCmd)
	appCmd.AddCommand(appStopCmd)
	appCmd.AddCommand(appRestartCmd)
	appCmd.AddCommand(appStatusCmd)
	rootCmd.AddCommand(appCmd)
}

func resolveInstalledAppID(input string) (string, error) {
	id := strings.TrimSpace(input)
	if id == "" {
		return "", fmt.Errorf("app_id is required")
	}

	apps, err := loadInstalledApps()
	if err != nil {
		return "", err
	}
	if len(apps) == 0 {
		return id, nil
	}

	for _, app := range apps {
		if app.ID == id {
			return app.ID, nil
		}
	}

	for _, app := range apps {
		if strings.EqualFold(strings.TrimSpace(app.Name), id) {
			return app.ID, nil
		}
	}

	matches := make([]string, 0)
	for _, app := range apps {
		if strings.HasPrefix(strings.ToLower(app.ID), strings.ToLower(id)) {
			matches = append(matches, app.ID)
		}
	}

	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		sort.Strings(matches)
		return "", fmt.Errorf("app_id %q is ambiguous. candidates: %s", id, strings.Join(matches, ", "))
	}

	return id, nil
}
