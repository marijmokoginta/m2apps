package cmd

import (
	"fmt"
	"m2apps/internal/config"
	"m2apps/internal/daemon"
	"m2apps/internal/downloader"
	"m2apps/internal/env"
	"m2apps/internal/github"
	"m2apps/internal/installer"
	"m2apps/internal/requirements"
	_ "m2apps/internal/requirements/checkers"
	"m2apps/internal/storage"
	"m2apps/internal/ui"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install application from install.json",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(ui.Info("[INFO] Reading install.json..."))

		cfg, err := config.LoadFromFile("install.json")
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			return
		}

		if err := cfg.Validate(); err != nil {
			fmt.Println(ui.Error("[ERROR] Error in install.json:"))
			fmt.Println(err)
			return
		}

		store, err := storage.New()
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			fmt.Println(ui.Error("[ERROR] Installation aborted."))
			os.Exit(1)
		}

		exists, err := store.Exists(cfg.AppID)
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			fmt.Println(ui.Error("[ERROR] Installation aborted."))
			os.Exit(1)
		}
		if exists {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] Application with app_id '%s' is already installed.", cfg.AppID)))
			fmt.Println(ui.Error("[ERROR] Installation aborted."))
			os.Exit(1)
		}

		channel := github.NormalizeChannel(cfg.Channel)

		fmt.Println(ui.Success("[OK] Config loaded"))
		fmt.Printf("%s %s\n", ui.Info("[INFO] App:"), cfg.Name)
		fmt.Printf("%s %s\n", ui.Info("[INFO] Preset:"), cfg.Preset)
		fmt.Printf("%s %s\n", ui.Info("[INFO] Channel:"), channel)

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
			fmt.Println(ui.Error("[ERROR] Installation aborted."))
			os.Exit(1)
		}

		owner, repo, err := github.ParseRepo(cfg.Source.Repo)
		if err != nil {
			fmt.Println()
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			fmt.Println(ui.Error("[ERROR] Installation aborted."))
			os.Exit(1)
		}

		ghClient := github.NewClient(cfg.Auth.Value)

		fmt.Println()
		fmt.Println(ui.Info(fmt.Sprintf("[INFO] Channel: %s", channel)))
		fmt.Println(ui.Info("[INFO] Resolving latest version..."))
		release, err := github.SelectLatestReleaseByChannel(ghClient, owner, repo, channel)
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			fmt.Println(ui.Error("[ERROR] Installation aborted."))
			os.Exit(1)
		}

		fmt.Printf("%s %s\n", ui.Success("[OK] Selected version:"), release.TagName)

		asset, err := github.FindAsset(release, cfg.Source.Asset)
		if err != nil {
			fmt.Println(ui.Error("[ERROR] Asset not found in selected release"))
			fmt.Println(ui.Error("[ERROR] Installation aborted."))
			os.Exit(1)
		}

		dest := filepath.Join(".", asset.Name)
		fmt.Println()
		fmt.Printf("%s %s...\n", ui.Info("[INFO] Downloading"), asset.Name)

		dl := downloader.New(cfg.Auth.Value)
		if err := dl.Download(asset.URL, dest, printDownloadProgress); err != nil {
			fmt.Println()
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			fmt.Println(ui.Error("[ERROR] Installation aborted."))
			os.Exit(1)
		}

		fmt.Println()
		fmt.Println(ui.Success("[OK] Download completed."))

		cwd, err := os.Getwd()
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			fmt.Println(ui.Error("[ERROR] Installation aborted."))
			os.Exit(1)
		}

		installCtx := installer.InstallContext{
			ZipPath:   dest,
			TargetDir: cwd,
			Preset:    cfg.Preset,
			AppID:     cfg.AppID,
		}

		if err := installer.Install(installCtx); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			fmt.Println(ui.Error("[ERROR] Installation aborted."))
			os.Exit(1)
		}

		appConfig := storage.AppConfig{
			AppID:       cfg.AppID,
			Name:        cfg.Name,
			InstallPath: cwd,
			Repo:        cfg.Source.Repo,
			Asset:       cfg.Source.Asset,
			Token:       cfg.Auth.Value,
			Version:     release.TagName,
			Channel:     channel,
			Preset:      cfg.Preset,
		}

		daemonManager, err := daemon.NewManager()
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			fmt.Println(ui.Error("[ERROR] Installation aborted."))
			os.Exit(1)
		}

		fmt.Println(ui.Info("[INFO] Ensuring daemon is running..."))
		if err := daemonManager.Start(); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			fmt.Println(ui.Error("[ERROR] Installation aborted."))
			os.Exit(1)
		}

		port, err := daemonManager.Port()
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			fmt.Println(ui.Error("[ERROR] Installation aborted."))
			os.Exit(1)
		}

		apiToken, err := daemon.GenerateAPIToken()
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			fmt.Println(ui.Error("[ERROR] Installation aborted."))
			os.Exit(1)
		}

		appConfig.APIToken = apiToken
		if err := store.Save(cfg.AppID, appConfig); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			fmt.Println(ui.Error("[ERROR] Installation aborted."))
			os.Exit(1)
		}

		apiURL := fmt.Sprintf("http://127.0.0.1:%d", port)
		if err := env.Inject(cwd, map[string]string{
			"M2APPS_API_URL":   apiURL,
			"M2APPS_API_TOKEN": apiToken,
			"M2APPS_APP_ID":    cfg.AppID,
		}); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			fmt.Println(ui.Error("[ERROR] Installation aborted."))
			os.Exit(1)
		}

		if err := daemonManager.RegisterApp(cfg.AppID); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			fmt.Println(ui.Error("[ERROR] Installation aborted."))
			os.Exit(1)
		}

		if err := os.Remove("install.json"); err != nil && !os.IsNotExist(err) {
			fmt.Println(ui.Warning(fmt.Sprintf("[WARN] Failed to remove install.json: %v", err)))
		}

		fmt.Println(ui.Success("[OK] Installation completed."))
	},
}

func printRequirementResults(results []requirements.Result) bool {
	fmt.Println()
	fmt.Println(ui.Info("[INFO] Checking requirements..."))
	fmt.Println()

	hasFailure := false

	for _, res := range results {
		label := formatRequirementLabel(res)

		if res.Success {
			fmt.Println(ui.Success(fmt.Sprintf("[OK] %s (found %s)", label, res.Found)))
			continue
		}

		hasFailure = true

		switch {
		case res.Found == "not found":
			fmt.Println(ui.Error(fmt.Sprintf("[FAIL] %s (not found)", label)))
		case strings.TrimSpace(res.Found) != "":
			fmt.Println(ui.Error(fmt.Sprintf("[FAIL] %s (found %s)", label, res.Found)))
		case strings.TrimSpace(res.Message) != "":
			fmt.Println(ui.Error(fmt.Sprintf("[FAIL] %s (%s)", label, res.Message)))
		default:
			fmt.Println(ui.Error(fmt.Sprintf("[FAIL] %s (failed)", label)))
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

func printDownloadProgress(read, total int64) {
	if total <= 0 {
		fmt.Printf("\rDownloaded %s", formatBytes(read))
		return
	}

	percent := int(float64(read) * 100 / float64(total))
	if percent > 100 {
		percent = 100
	}
	if percent < 0 {
		percent = 0
	}

	const barWidth = 10
	filled := percent * barWidth / 100
	if filled > barWidth {
		filled = barWidth
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	fmt.Printf("\r[%s] %d%% (%s / %s)", bar, percent, formatBytes(read), formatBytes(total))
}

func formatBytes(size int64) string {
	if size < 1024 {
		return strconv.FormatInt(size, 10) + "B"
	}

	kb := float64(size) / 1024
	if kb < 1024 {
		return fmt.Sprintf("%.1fKB", kb)
	}

	mb := kb / 1024
	if mb < 1024 {
		return fmt.Sprintf("%.1fMB", mb)
	}

	gb := mb / 1024
	return fmt.Sprintf("%.1fGB", gb)
}
