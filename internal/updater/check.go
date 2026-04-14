package updater

import (
	"fmt"
	"m2apps/internal/github"
	"m2apps/internal/storage"
)

type CheckResult struct {
	HasUpdate      bool   `json:"has_update"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
}

func Check(appID string) (CheckResult, error) {
	store, err := storage.New()
	if err != nil {
		return CheckResult{}, err
	}

	config, err := store.Load(appID)
	if err != nil {
		return CheckResult{}, fmt.Errorf("failed to load app metadata: %w", err)
	}

	owner, repo, err := github.ParseRepo(config.Repo)
	if err != nil {
		return CheckResult{}, err
	}

	channel := github.NormalizeChannel(config.Channel)
	client := github.NewClient(config.Token)
	target, err := github.SelectLatestReleaseByChannel(client, owner, repo, channel)
	if err != nil {
		return CheckResult{}, err
	}

	newer, err := IsNewer(target.TagName, config.Version)
	if err != nil {
		return CheckResult{}, fmt.Errorf("failed to compare versions: %w", err)
	}

	return CheckResult{
		HasUpdate:      newer,
		CurrentVersion: config.Version,
		LatestVersion:  target.TagName,
	}, nil
}
