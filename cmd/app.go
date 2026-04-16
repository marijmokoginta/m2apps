package cmd

import (
	"fmt"
	"m2apps/internal/network"
	"m2apps/internal/privilege"
	"m2apps/internal/process"
	"m2apps/internal/ui"
	"os"
	"sort"
	"strconv"
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
		if err := runAppCommand("start", args[0]); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}
	},
}

var appStopCmd = &cobra.Command{
	Use:   "stop <app_id>",
	Short: "Stop app processes",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runAppCommand("stop", args[0]); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}
	},
}

var appRestartCmd = &cobra.Command{
	Use:   "restart <app_id>",
	Short: "Restart app processes",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runAppCommand("restart", args[0]); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}
	},
}

var appStatusCmd = &cobra.Command{
	Use:   "status <app_id>",
	Short: "Show app process status",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runAppCommand("status", args[0]); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
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

func runAppCommand(action string, inputAppID string) error {
	appID, err := resolveInstalledAppID(inputAppID)
	if err != nil {
		return err
	}

	if appID != strings.TrimSpace(inputAppID) {
		fmt.Println(ui.Info(fmt.Sprintf("[INFO] Using installed app_id: %s", appID)))
	}

	manager, err := process.NewManager()
	if err != nil {
		return err
	}

	switch strings.ToLower(strings.TrimSpace(action)) {
	case "start":
		fmt.Println(ui.Info(fmt.Sprintf("[INFO] Starting app %s...", appID)))
		state, err := manager.Start(appID)
		if err != nil {
			return err
		}
		fmt.Println(ui.Success("[OK] App started"))
		fmt.Println(ui.Info(fmt.Sprintf("[INFO] Started %d process(es) for %s", len(state.Processes), appID)))
		for _, proc := range state.Processes {
			url := inferProcessURL(proc)
			if url != "-" {
				fmt.Printf("- %s | pid: %d | url: %s\n", proc.Name, proc.PID, url)
				continue
			}
			fmt.Printf("- %s | pid: %d\n", proc.Name, proc.PID)
		}
		if appURL := inferAppURL(state.Processes); appURL != "-" {
			fmt.Println(ui.Info(fmt.Sprintf("URL: %s", appURL)))
		}
		return nil

	case "stop":
		fmt.Println(ui.Info(fmt.Sprintf("[INFO] Stopping app %s...", appID)))
		state, err := manager.Stop(appID)
		if err != nil {
			if escalated, escalateErr := retryAppActionWithElevation("stop", appID, err); escalated {
				return escalateErr
			}
			return err
		}
		if len(state.Processes) == 0 {
			fmt.Println(ui.Warning(fmt.Sprintf("[WARN] No process records found for %s", appID)))
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("[OK] Stopped app %s", appID)))
		return nil

	case "restart":
		fmt.Println(ui.Info(fmt.Sprintf("[INFO] Restarting app %s...", appID)))
		state, err := manager.Restart(appID)
		if err != nil {
			if escalated, escalateErr := retryAppActionWithElevation("restart", appID, err); escalated {
				return escalateErr
			}
			return err
		}
		fmt.Println(ui.Success(fmt.Sprintf("[OK] Restarted app %s (%d process(es))", appID, len(state.Processes))))
		return nil

	case "status":
		state, err := manager.Status(appID)
		if err != nil {
			return err
		}
		if len(state.Processes) == 0 {
			fmt.Println(ui.Warning(fmt.Sprintf("[WARN] No process records found for %s", appID)))
			return nil
		}

		fmt.Println(ui.Info(fmt.Sprintf("[INFO] App %s process status:", appID)))
		fmt.Println(renderProcessStatusTable(state.Processes))
		return nil

	default:
		return fmt.Errorf("unsupported app process action %q", action)
	}
}

func retryAppActionWithElevation(action, appID string, originalErr error) (bool, error) {
	if originalErr == nil {
		return false, nil
	}
	if privilege.IsElevated() || !shouldAttemptElevation(action, originalErr) {
		return false, nil
	}

	fmt.Println(ui.Info("[INFO] Supervisor permission required. Opening OS authentication popup..."))
	if err := privilege.RelaunchElevated([]string{"app", strings.TrimSpace(action), strings.TrimSpace(appID)}); err != nil {
		return true, err
	}

	return true, nil
}

func isPrivilegeError(err error) bool {
	if err == nil {
		return false
	}

	text := strings.ToLower(strings.TrimSpace(err.Error()))
	if text == "" {
		return false
	}

	return strings.Contains(text, "permission denied") ||
		strings.Contains(text, "operation not permitted") ||
		strings.Contains(text, "access is denied")
}

func shouldAttemptElevation(action string, err error) bool {
	if isPrivilegeError(err) {
		return true
	}

	act := strings.ToLower(strings.TrimSpace(action))
	if act != "stop" && act != "restart" {
		return false
	}

	text := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(text, "failed to stop pid") ||
		strings.Contains(text, "process is still running")
}

func renderProcessStatusTable(processes []process.Process) string {
	headers := []string{"NAME", "PID", "STATUS", "URL", "COMMAND"}
	rows := make([][]string, 0, len(processes))

	widths := []int{
		len(headers[0]),
		len(headers[1]),
		len(headers[2]),
		len(headers[3]),
		len(headers[4]),
	}

	for _, proc := range processes {
		command := strings.Join(proc.Command, " ")
		if command == "" {
			command = "-"
		}

		row := []string{
			proc.Name,
			strconv.Itoa(proc.PID),
			proc.Status,
			inferProcessURL(proc),
			command,
		}
		rows = append(rows, row)

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

	formatRow := func(cols []string) string {
		cells := make([]string, 0, len(cols))
		for i, col := range cols {
			padded := fmt.Sprintf("%-*s", widths[i], col)
			if i == 2 {
				padded = colorizeProcessStatusCell(col, padded)
			}
			cells = append(cells, " "+padded+" ")
		}
		return "|" + strings.Join(cells, "|") + "|"
	}

	lines := []string{
		border(),
		formatRow(headers),
		border(),
	}

	for _, row := range rows {
		lines = append(lines, formatRow(row))
	}
	lines = append(lines, border())

	return strings.Join(lines, "\n")
}

func colorizeProcessStatusCell(rawStatus string, paddedStatus string) string {
	status := strings.ToLower(strings.TrimSpace(rawStatus))

	switch status {
	case "running", "started", "start", "up":
		return ui.Success(paddedStatus)
	case "stopped", "stop", "dead", "failed", "down":
		return ui.Error(paddedStatus)
	default:
		return ui.Warning(paddedStatus)
	}
}

func inferProcessURL(proc process.Process) string {
	if !isWebOrServerProcess(proc.Name) {
		return "-"
	}

	host, parsedPort := extractHostPort(proc.Command)
	port := proc.Port
	if port <= 0 && parsedPort != "" {
		port = atoiOrZero(parsedPort)
	}
	if port <= 0 {
		return "-"
	}

	if host == "" || host == "0.0.0.0" || host == "::" {
		lanIP, err := network.ResolveLocalIPv4()
		if err == nil && lanIP != "" {
			host = lanIP
		} else {
			host = "127.0.0.1"
		}
	}

	return fmt.Sprintf("http://%s:%d", host, port)
}

func isWebOrServerProcess(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	return normalized == "web" || normalized == "server"
}

func extractHostPort(command []string) (string, string) {
	host := ""
	port := ""

	for i := 0; i < len(command); i++ {
		arg := strings.TrimSpace(command[i])
		lower := strings.ToLower(arg)
		if arg == "" {
			continue
		}

		switch {
		case strings.HasPrefix(lower, "--host="):
			host = strings.TrimSpace(arg[len("--host="):])
		case lower == "--host" && i+1 < len(command):
			host = strings.TrimSpace(command[i+1])
			i++
		case strings.HasPrefix(lower, "--port="):
			port = normalizePortToken(strings.TrimSpace(arg[len("--port="):]))
		case lower == "--port" && i+1 < len(command):
			port = normalizePortToken(strings.TrimSpace(command[i+1]))
			i++
		case strings.HasPrefix(lower, "-p="):
			port = normalizePortToken(strings.TrimSpace(arg[len("-p="):]))
		case lower == "-p" && i+1 < len(command):
			port = normalizePortToken(strings.TrimSpace(command[i+1]))
			i++
		case strings.HasPrefix(strings.ToUpper(arg), "PORT="):
			port = normalizePortToken(strings.TrimSpace(arg[len("PORT="):]))
		}
	}

	if port == "" && isLaravelServeCommand(command) {
		port = "8000"
	}

	if host == "" && port != "" {
		host = "127.0.0.1"
	}

	if strings.Contains(host, ":") && port == "" {
		parts := strings.Split(host, ":")
		host = strings.TrimSpace(parts[0])
		port = normalizePortToken(strings.TrimSpace(parts[len(parts)-1]))
	}

	return host, port
}

func normalizePortToken(input string) string {
	value := strings.TrimSpace(input)
	if value == "" {
		return ""
	}

	digits := make([]rune, 0, len(value))
	for _, ch := range value {
		if ch >= '0' && ch <= '9' {
			digits = append(digits, ch)
		}
	}
	return string(digits)
}

func inferAppURL(processes []process.Process) string {
	for _, proc := range processes {
		if url := inferProcessURL(proc); url != "-" {
			return url
		}
	}

	for _, proc := range processes {
		if proc.Port <= 0 {
			continue
		}

		host, _ := extractHostPort(proc.Command)
		if host == "" || host == "0.0.0.0" || host == "::" {
			host = "127.0.0.1"
		}

		return fmt.Sprintf("http://%s:%d", host, proc.Port)
	}

	return "-"
}

func atoiOrZero(input string) int {
	value := strings.TrimSpace(input)
	if value == "" {
		return 0
	}
	number, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return number
}

func isLaravelServeCommand(command []string) bool {
	if len(command) < 3 {
		return false
	}

	return strings.EqualFold(strings.TrimSpace(command[0]), "php") &&
		strings.EqualFold(strings.TrimSpace(command[1]), "artisan") &&
		strings.EqualFold(strings.TrimSpace(command[2]), "serve")
}
