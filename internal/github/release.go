package github

import (
	"fmt"
	"strings"
)

func ParseRepo(input string) (owner string, repo string, err error) {
	parts := strings.Split(strings.TrimSpace(input), "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repo format, expected owner/repo")
	}

	owner = strings.TrimSpace(parts[0])
	repo = strings.TrimSpace(parts[1])
	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("invalid repo format, expected owner/repo")
	}

	return owner, repo, nil
}

func FindAsset(release *Release, name string) (*Asset, error) {
	if release == nil {
		return nil, fmt.Errorf("release data is empty")
	}

	target := strings.TrimSpace(name)
	for i := range release.Assets {
		if release.Assets[i].Name == target {
			if strings.TrimSpace(release.Assets[i].URL) == "" {
				return nil, fmt.Errorf("asset %q is missing API url", target)
			}
			return &release.Assets[i], nil
		}
	}

	return nil, fmt.Errorf("asset %q not found in release %s", target, release.TagName)
}
