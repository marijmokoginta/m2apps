package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"m2apps/internal/selfupdate"
	"m2apps/internal/storage"
	"m2apps/internal/system"
	"m2apps/internal/ui"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

const colorReset = "\033[0m"
const appVersion = "v1.1.5"

func rgb(r, g, b int) string {
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func gradientLine(s string, sr, sg, sb, er, eg, eb int) string {
	runes := []rune(s)
	n := len(runes)
	if n == 0 {
		return colorReset
	}
	denom := n - 1
	if denom < 1 {
		denom = 1
	}
	var buf strings.Builder
	buf.Grow(n * 20)
	for i, ch := range runes {
		t := float64(i) / float64(denom)
		r := clamp(int(math.Round(float64(sr)+t*float64(er-sr))), 0, 255)
		g := clamp(int(math.Round(float64(sg)+t*float64(eg-sg))), 0, 255)
		b := clamp(int(math.Round(float64(sb)+t*float64(eb-sb))), 0, 255)
		buf.WriteString(rgb(r, g, b))
		buf.WriteRune(ch)
	}
	buf.WriteString(colorReset)
	return buf.String()
}

func gradientLineShifted(s string, phase float64) string {
	runes := []rune(s)
	n := len(runes)
	if n == 0 {
		return colorReset
	}
	denom := n - 1
	if denom < 1 {
		denom = 1
	}
	var buf strings.Builder
	buf.Grow(n * 20)
	for i, ch := range runes {
		raw := float64(i)/float64(denom) + phase*0.35
		t := math.Mod(math.Abs(raw), 2.0)
		if t > 1.0 {
			t = 2.0 - t
		}

		var r, g, b int
		if t < 0.5 {
			u := t / 0.5
			r = clamp(int(math.Round(0+u*120)), 0, 255)
			g = clamp(int(math.Round(220+u*(80-220))), 0, 255)
			b = clamp(int(math.Round(255+u*(255-255))), 0, 255)
		} else {
			u := (t - 0.5) / 0.5
			r = clamp(int(math.Round(120+u*(255-120))), 0, 255)
			g = clamp(int(math.Round(80+u*(50-80))), 0, 255)
			b = clamp(int(math.Round(255+u*(200-255))), 0, 255)
		}
		buf.WriteString(rgb(r, g, b))
		buf.WriteRune(ch)
	}
	buf.WriteString(colorReset)
	return buf.String()
}

var bannerLines = []string{
	` __  __ ___   ____          _        _                      `,
	`|  \/  |__ \ / ___|___   __| | ___  / \   _ __  _ __  ___  `,
	`| |\/| | / /| |   / _ \ / _` + "`" + `/ _ \/ _ \ | '_ \| '_ \/ __| `,
	`| |  | |/ /_| |__| (_) | (_| |  __/ ___ \| |_) | |_) \__ \ `,
	`|_|  |_|____|\____\___/ \__,_|\___/_/   \_\ .__/| .__/|___/ `,
	`                                          |_|   |_|         `,
}

func printBanner() {
	totalLines := float64(len(bannerLines) - 1)
	for i, line := range bannerLines {
		phase := float64(i) / totalLines
		fmt.Println(gradientLineShifted(line, phase))
	}

	subtitle := "  Auto Updater Engine"
	fmt.Println(gradientLine(subtitle, 255, 220, 0, 255, 140, 0))

	author := "  by Marij Mokoginta"
	fmt.Println(gradientLine(author, 255, 200, 180, 255, 100, 80))

	fmt.Println()
}

var rootCmd = &cobra.Command{
	Use:     "m2apps",
	Short:   "M2Apps CLI",
	Version: appVersion,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runInteractiveRoot(cmd); err != nil {
			if errors.Is(err, ui.ErrMenuCancelled) {
				return
			}

			fmt.Println(ui.Warning(fmt.Sprintf("[WARN] Interactive mode unavailable: %v", err)))
			_ = cmd.Help()
		}
	},
}

const menuActionBack = "__back__"

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(listCmd)
}

func runInteractiveRoot(cmd *cobra.Command) error {
	shouldExit, err := runSelfUpdateFlow()
	if err != nil {
		fmt.Println(ui.Warning(fmt.Sprintf("[WARN] Self-update check skipped: %v", err)))
	} else if shouldExit {
		return nil
	}

	for {
		action, err := runInteractiveMenu(
			"Main Menu",
			[]ui.MenuItem{
				{Title: "Install Application", Action: "install"},
				{Title: "Update Application", Action: "update"},
				{Title: "Manage Application Process", Action: "process"},
				{Title: "Delete Application", Action: "delete"},
				{Title: "Switch Channel", Action: "channel"},
				{Title: "List Installed Applications", Action: "list"},
				{Title: "Manage Daemon Service", Action: "daemon"},
				{Title: "Help", Action: "help"},
				{Title: "Exit", Action: "exit"},
			},
			nil,
		)
		if err != nil {
			if errors.Is(err, ui.ErrMenuCancelled) {
				return nil
			}
			return err
		}

		switch action {
		case "install":
			installCmd.Run(installCmd, nil)
			promptBackToMainMenu()
		case "update":
			if err := runInteractiveUpdateFlow(); err != nil {
				fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
				promptBackToMainMenu()
			}
		case "process":
			if err := runInteractiveProcessFlow(); err != nil {
				fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
				promptBackToMainMenu()
			}
		case "delete":
			if err := runInteractiveDeleteFlow(); err != nil {
				fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
				promptBackToMainMenu()
			}
		case "channel":
			if err := runInteractiveChannelFlow(); err != nil {
				fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
				promptBackToMainMenu()
			}
		case "list":
			listCmd.Run(listCmd, nil)
			promptBackToMainMenu()
		case "daemon":
			if err := runInteractiveDaemonFlow(); err != nil {
				fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
				promptBackToMainMenu()
			}
		case "help":
			_ = cmd.Help()
			promptBackToMainMenu()
		case "exit":
			return nil
		}
	}
}

func runSelfUpdateFlow() (bool, error) {
	check, err := selfupdate.Check(appVersion)
	if err != nil {
		return false, err
	}
	if !check.HasUpdate || check.Skipped {
		return false, nil
	}

	title := fmt.Sprintf("New version available: %s", check.LatestVersion)
	action, err := runInteractiveMenu(
		title,
		[]ui.MenuItem{
			{Title: "Update now", Action: "update_now"},
			{Title: "Skip for now", Action: "skip_once"},
			{Title: "Skip until next version", Action: "skip_until"},
		},
		nil,
	)
	if err != nil {
		if errors.Is(err, ui.ErrMenuCancelled) {
			return false, nil
		}
		return false, err
	}

	switch action {
	case "update_now":
		fmt.Println(ui.Info(fmt.Sprintf("[INFO] Updating m2apps from %s to %s...", check.CurrentVersion, check.LatestVersion)))
		if err := selfupdate.Update(appVersion); err != nil {
			if errors.Is(err, selfupdate.ErrRestartScheduled) {
				fmt.Println(ui.Success("[OK] Update applied. Restarting m2apps..."))
				fmt.Println(ui.Info("[INFO] Press Enter to continue and close current process..."))
				reader := bufio.NewReader(os.Stdin)
				_, _ = reader.ReadString('\n')
				return true, nil
			}
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] Self-update failed: %v", err)))
			promptBackToMainMenu()
			return false, nil
		}
		fmt.Println(ui.Success("[OK] Self-update completed."))
		fmt.Println(ui.Info("[INFO] Restart m2apps to use the new version."))
		promptBackToMainMenu()
		return false, nil
	case "skip_once":
		return false, nil
	case "skip_until":
		if err := selfupdate.SaveSkippedVersion(check.LatestVersion); err != nil {
			return false, err
		}
		fmt.Println(ui.Info(fmt.Sprintf("[INFO] Skipped until version %s", check.LatestVersion)))
		return false, nil
	default:
		return false, nil
	}
}

type installedApp struct {
	ID          string
	Name        string
	Version     string
	Channel     string
	Preset      string
	InstallPath string
}

func runInteractiveUpdateFlow() error {
	apps, err := loadInstalledApps()
	if err != nil {
		return err
	}
	if len(apps) == 0 {
		fmt.Println(ui.Warning("[WARN] No installed applications found."))
		promptBackToMainMenu()
		return nil
	}

	appID, err := runInteractiveMenu("Select Application to Update", withBackMenuItems(toAppMenuItems(apps), "Back"), nil)
	if err != nil {
		if errors.Is(err, ui.ErrMenuCancelled) {
			return nil
		}
		return err
	}
	if appID == menuActionBack {
		return nil
	}

	if err := runUpdate(appID); err != nil {
		return fmt.Errorf("failed to update app: %w", err)
	}

	promptBackToMainMenu()
	return nil
}

func runInteractiveChannelFlow() error {
	for {
		apps, err := loadInstalledApps()
		if err != nil {
			return err
		}
		if len(apps) == 0 {
			fmt.Println(ui.Warning("[WARN] No installed applications found."))
			promptBackToMainMenu()
			return nil
		}

		appID, err := runInteractiveMenu("Select Application", withBackMenuItems(toAppMenuItems(apps), "Back"), nil)
		if err != nil {
			if errors.Is(err, ui.ErrMenuCancelled) {
				return nil
			}
			return err
		}
		if appID == menuActionBack {
			return nil
		}

		channel, err := runInteractiveMenu(
			"Select Channel",
			withBackMenuItems(
				[]ui.MenuItem{
					{Title: "stable", Action: "stable"},
					{Title: "beta", Action: "beta"},
					{Title: "alpha", Action: "alpha"},
				},
				"Back",
			),
			nil,
		)
		if err != nil {
			if errors.Is(err, ui.ErrMenuCancelled) {
				continue
			}
			return err
		}
		if channel == menuActionBack {
			continue
		}

		message, err := runSetChannel(appID, channel)
		if err != nil {
			return err
		}
		fmt.Println(ui.Success(message))
		promptBackToMainMenu()
		return nil
	}
}

func runInteractiveDeleteFlow() error {
	apps, err := loadInstalledApps()
	if err != nil {
		return err
	}
	if len(apps) == 0 {
		fmt.Println(ui.Warning("[WARN] No installed applications found."))
		promptBackToMainMenu()
		return nil
	}

	appID, err := runInteractiveMenu("Select Application to Delete", withBackMenuItems(toAppMenuItems(apps), "Back"), nil)
	if err != nil {
		if errors.Is(err, ui.ErrMenuCancelled) {
			return nil
		}
		return err
	}
	if appID == menuActionBack {
		return nil
	}

	if err := runDelete(appID); err != nil {
		return fmt.Errorf("failed to delete app: %w", err)
	}

	fmt.Println(ui.Success(fmt.Sprintf("[OK] Application %s deleted", appID)))
	promptBackToMainMenu()
	return nil
}

func runInteractiveProcessFlow() error {
	for {
		apps, err := loadInstalledApps()
		if err != nil {
			return err
		}
		if len(apps) == 0 {
			fmt.Println(ui.Warning("[WARN] No installed applications found."))
			promptBackToMainMenu()
			return nil
		}

		appID, err := runInteractiveMenu("Select Application", withBackMenuItems(toAppMenuItems(apps), "Back"), nil)
		if err != nil {
			if errors.Is(err, ui.ErrMenuCancelled) {
				return nil
			}
			return err
		}
		if appID == menuActionBack {
			return nil
		}

		action, err := runInteractiveMenu(
			"Select Process Action",
			withBackMenuItems(
				[]ui.MenuItem{
					{Title: "Start", Action: "start"},
					{Title: "Stop", Action: "stop"},
					{Title: "Restart", Action: "restart"},
					{Title: "Status", Action: "status"},
				},
				"Back",
			),
			nil,
		)
		if err != nil {
			if errors.Is(err, ui.ErrMenuCancelled) {
				continue
			}
			return err
		}
		if action == menuActionBack {
			continue
		}

		if err := runAppCommand(action, appID); err != nil {
			return fmt.Errorf("failed to manage app process: %w", err)
		}

		promptBackToMainMenu()
		return nil
	}
}

func runInteractiveDaemonFlow() error {
	for {
		action, err := runInteractiveMenu(
			"Daemon Service",
			withBackMenuItems(
				[]ui.MenuItem{
					{Title: "Install Service", Action: "install"},
					{Title: "Uninstall Service", Action: "uninstall"},
					{Title: "Enable Service", Action: "enable"},
					{Title: "Disable Service", Action: "disable"},
					{Title: "Start Service", Action: "start"},
					{Title: "Stop Service", Action: "stop"},
					{Title: "Status Service", Action: "status"},
					{Title: "Local API Diagnostics", Action: "api_menu"},
				},
				"Back",
			),
			nil,
		)
		if err != nil {
			if errors.Is(err, ui.ErrMenuCancelled) {
				return nil
			}
			return err
		}
		if action == menuActionBack {
			return nil
		}

		if action == "api_menu" {
			if err := runInteractiveLocalAPIFlow(); err != nil {
				return err
			}
			return nil
		}

		if err := runDaemonCommand(action); err != nil {
			return err
		}

		promptBackToMainMenu()
		return nil
	}
}

func runInteractiveLocalAPIFlow() error {
	for {
		action, err := runInteractiveMenu(
			"Local API Diagnostics",
			withBackMenuItems(
				[]ui.MenuItem{
					{Title: "Show Local API Detail", Action: "api_info"},
					{Title: "Ping Local API", Action: "api_ping"},
				},
				"Back",
			),
			nil,
		)
		if err != nil {
			if errors.Is(err, ui.ErrMenuCancelled) {
				return nil
			}
			return err
		}
		if action == menuActionBack {
			return nil
		}

		if err := runDaemonCommand(action); err != nil {
			return err
		}

		promptBackToMainMenu()
		return nil
	}
}

func loadInstalledApps() ([]installedApp, error) {
	store, err := storage.New()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(system.GetAppsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return []installedApp{}, nil
		}
		return nil, fmt.Errorf("failed to read installed apps: %w", err)
	}

	apps := make([]installedApp, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		appID := strings.TrimSpace(entry.Name())
		if appID == "" {
			continue
		}

		cfg, err := store.Load(appID)
		if err != nil {
			apps = append(apps, installedApp{
				ID:          appID,
				Name:        appID,
				Version:     "-",
				Channel:     "-",
				Preset:      "-",
				InstallPath: "-",
			})
			continue
		}

		name := strings.TrimSpace(cfg.Name)
		if name == "" {
			name = appID
		}

		apps = append(apps, installedApp{
			ID:          appID,
			Name:        name,
			Version:     strings.TrimSpace(cfg.Version),
			Channel:     strings.TrimSpace(cfg.Channel),
			Preset:      strings.TrimSpace(cfg.Preset),
			InstallPath: strings.TrimSpace(cfg.InstallPath),
		})
	}

	sort.Slice(apps, func(i, j int) bool {
		return apps[i].ID < apps[j].ID
	})

	return apps, nil
}

func toAppMenuItems(apps []installedApp) []ui.MenuItem {
	items := make([]ui.MenuItem, 0, len(apps))
	for _, app := range apps {
		title := app.ID
		if app.Name != "" && app.Name != app.ID {
			title = fmt.Sprintf("%s (%s)", app.Name, app.ID)
		}

		items = append(items, ui.MenuItem{
			Title:  title,
			Action: app.ID,
		})
	}
	return items
}

func withBackMenuItems(items []ui.MenuItem, backTitle string) []ui.MenuItem {
	cloned := make([]ui.MenuItem, 0, len(items)+1)
	cloned = append(cloned, items...)
	cloned = append(cloned, ui.MenuItem{
		Title:  backTitle,
		Action: menuActionBack,
	})
	return cloned
}

func promptBackToMainMenu() {
	fmt.Println()
	fmt.Println(ui.Info("[INFO] Press Enter to back to Main Menu..."))
	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')
}

func runInteractiveMenu(title string, items []ui.MenuItem, staticItems []string) (string, error) {
	refreshInteractiveScreen()
	return ui.RunMenu(title, items, staticItems)
}

func refreshInteractiveScreen() {
	fmt.Print("\033[H\033[2J")
	printBanner()
	fmt.Printf("%s %s\n\n", ui.Info("[INFO] Version:"), appVersion)
}
