package cmd

import (
	"errors"
	"fmt"
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
const appVersion = "v1.0.4"

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
		printBanner()
		fmt.Printf("%s %s\n\n", ui.Info("[INFO] Version:"), appVersion)

		if err := runInteractiveRoot(cmd); err != nil {
			if errors.Is(err, ui.ErrMenuCancelled) {
				return
			}

			fmt.Println(ui.Warning(fmt.Sprintf("[WARN] Interactive mode unavailable: %v", err)))
			_ = cmd.Help()
		}
	},
}

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
	action, err := ui.RunMenu(
		"Main Menu",
		[]ui.MenuItem{
			{Title: "Install Application", Action: "install"},
			{Title: "Update Application", Action: "update"},
			{Title: "Delete Application", Action: "delete"},
			{Title: "Switch Channel", Action: "channel"},
		},
		[]string{"list", "daemon", "help"},
	)
	if err != nil {
		return err
	}

	switch action {
	case "install":
		installCmd.Run(installCmd, nil)
		return nil
	case "update":
		return runInteractiveUpdateFlow()
	case "delete":
		return runInteractiveDeleteFlow()
	case "channel":
		return runInteractiveChannelFlow()
	default:
		return nil
	}
}

type installedApp struct {
	ID   string
	Name string
}

func runInteractiveUpdateFlow() error {
	apps, err := loadInstalledApps()
	if err != nil {
		return err
	}
	if len(apps) == 0 {
		fmt.Println(ui.Warning("[WARN] No installed applications found."))
		return nil
	}

	menuItems := toAppMenuItems(apps)
	appID, err := ui.RunMenu("Select Application to Update", menuItems, nil)
	if err != nil {
		if errors.Is(err, ui.ErrMenuCancelled) {
			return nil
		}
		return err
	}

	if err := runUpdate(appID); err != nil {
		return fmt.Errorf("failed to update app: %w", err)
	}

	return nil
}

func runInteractiveChannelFlow() error {
	apps, err := loadInstalledApps()
	if err != nil {
		return err
	}
	if len(apps) == 0 {
		fmt.Println(ui.Warning("[WARN] No installed applications found."))
		return nil
	}

	appID, err := ui.RunMenu("Select Application", toAppMenuItems(apps), nil)
	if err != nil {
		if errors.Is(err, ui.ErrMenuCancelled) {
			return nil
		}
		return err
	}

	channel, err := ui.RunMenu(
		"Select Channel",
		[]ui.MenuItem{
			{Title: "stable", Action: "stable"},
			{Title: "beta", Action: "beta"},
			{Title: "alpha", Action: "alpha"},
		},
		nil,
	)
	if err != nil {
		if errors.Is(err, ui.ErrMenuCancelled) {
			return nil
		}
		return err
	}

	message, err := runSetChannel(appID, channel)
	if err != nil {
		return err
	}
	fmt.Println(ui.Success(message))
	return nil
}

func runInteractiveDeleteFlow() error {
	apps, err := loadInstalledApps()
	if err != nil {
		return err
	}
	if len(apps) == 0 {
		fmt.Println(ui.Warning("[WARN] No installed applications found."))
		return nil
	}

	appID, err := ui.RunMenu("Select Application to Delete", toAppMenuItems(apps), nil)
	if err != nil {
		if errors.Is(err, ui.ErrMenuCancelled) {
			return nil
		}
		return err
	}

	if err := runDelete(appID); err != nil {
		return fmt.Errorf("failed to delete app: %w", err)
	}

	fmt.Println(ui.Success(fmt.Sprintf("[OK] Application %s deleted", appID)))
	return nil
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
				ID:   appID,
				Name: appID,
			})
			continue
		}

		name := strings.TrimSpace(cfg.Name)
		if name == "" {
			name = appID
		}

		apps = append(apps, installedApp{
			ID:   appID,
			Name: name,
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
