package cmd

import (
	"fmt"
	"m2apps/internal/config"
	"m2apps/internal/downloader"
	"m2apps/internal/github"
	"m2apps/internal/requirements"
	_ "m2apps/internal/requirements/checkers"
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
			fmt.Println("Installation aborted.")
			os.Exit(1)
		}

		owner, repo, err := github.ParseRepo(cfg.Source.Repo)
		if err != nil {
			fmt.Println()
			fmt.Println("Error:", err)
			fmt.Println("Installation aborted.")
			os.Exit(1)
		}

		ghClient := github.NewClient(cfg.Auth.Value)

		fmt.Println()
		release, err := fetchRelease(ghClient, owner, repo, cfg.Source.Version)
		if err != nil {
			fmt.Println("Error:", err)
			fmt.Println("Installation aborted.")
			os.Exit(1)
		}

		fmt.Printf("Found version: %s\n", release.TagName)

		asset, err := github.FindAsset(release, cfg.Source.Asset)
		if err != nil {
			fmt.Println("Error:", err)
			fmt.Println("Installation aborted.")
			os.Exit(1)
		}

		dest := filepath.Join(".", asset.Name)
		fmt.Println()
		fmt.Printf("Downloading %s...\n", asset.Name)

		dl := downloader.New(cfg.Auth.Value)
		if err := dl.Download(asset.URL, dest, printDownloadProgress); err != nil {
			fmt.Println()
			fmt.Println("Error:", err)
			fmt.Println("Installation aborted.")
			os.Exit(1)
		}

		fmt.Println()
		fmt.Println("Download completed.")
	},
}

func printRequirementResults(results []requirements.Result) bool {
	fmt.Println()
	fmt.Println("Checking requirements...")
	fmt.Println()

	hasFailure := false

	for _, res := range results {
		label := formatRequirementLabel(res)

		if res.Success {
			fmt.Printf("[✓] %s (found %s)\n", label, res.Found)
			continue
		}

		hasFailure = true

		switch {
		case res.Found == "not found":
			fmt.Printf("[✗] %s (not found)\n", label)
		case strings.TrimSpace(res.Found) != "":
			fmt.Printf("[✗] %s (found %s)\n", label, res.Found)
		case strings.TrimSpace(res.Message) != "":
			fmt.Printf("[✗] %s (%s)\n", label, res.Message)
		default:
			fmt.Printf("[✗] %s (failed)\n", label)
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

func fetchRelease(client github.Client, owner, repo, version string) (*github.Release, error) {
	if strings.EqualFold(strings.TrimSpace(version), "latest") {
		fmt.Println("Fetching latest release...")
		return client.GetLatestRelease(owner, repo)
	}

	fmt.Printf("Fetching release tag %s...\n", version)
	return client.GetReleaseByTag(owner, repo, version)
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
