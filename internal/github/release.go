package github

import (
	"fmt"
	"regexp"
	"strings"
)

var assetVersionPattern = regexp.MustCompile(`v?\d+\.\d+(?:\.\d+)?(?:-[0-9A-Za-z.-]+)?`)

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

// FindAssetByVersionedName resolves assets whose file name includes the release tag.
//
// Example:
// - configured: "app-v1.0.0.zip"
// - release tag: "v1.2.0"
// It will try to resolve "app-v1.2.0.zip" in the selected release.
//
// If configured does not contain a version-like substring, this behaves like FindAsset.
func FindAssetByVersionedName(release *Release, configured string) (*Asset, error) {
	if release == nil {
		return nil, fmt.Errorf("release data is empty")
	}

	cfg := strings.TrimSpace(configured)
	if cfg == "" {
		return nil, fmt.Errorf("asset name is required")
	}

	// First try exact match for backwards compatibility.
	if asset, err := FindAsset(release, cfg); err == nil {
		return asset, nil
	}

	loc := assetVersionPattern.FindStringIndex(cfg)
	if loc == nil {
		return nil, fmt.Errorf("asset %q not found in release %s", cfg, release.TagName)
	}

	prefix := cfg[:loc[0]]
	suffix := cfg[loc[1]:]

	tag := strings.TrimSpace(release.TagName)
	if tag != "" {
		derived := prefix + tag + suffix
		if asset, err := FindAsset(release, derived); err == nil {
			return asset, nil
		}
	}

	tagNoV := strings.TrimPrefix(tag, "v")
	if tagNoV != tag && tagNoV != "" {
		derived := prefix + tagNoV + suffix
		if asset, err := FindAsset(release, derived); err == nil {
			return asset, nil
		}
	}

	// Fallback: prefix/suffix match and contains the tag.
	candidates := make([]*Asset, 0)
	needle := strings.ToLower(tag)
	needleNoV := strings.ToLower(tagNoV)

	for i := range release.Assets {
		name := release.Assets[i].Name
		if prefix != "" && !strings.HasPrefix(name, prefix) {
			continue
		}
		if suffix != "" && !strings.HasSuffix(name, suffix) {
			continue
		}

		lower := strings.ToLower(name)
		if needle != "" && strings.Contains(lower, needle) {
			candidates = append(candidates, &release.Assets[i])
			continue
		}
		if needleNoV != "" && strings.Contains(lower, needleNoV) {
			candidates = append(candidates, &release.Assets[i])
			continue
		}
	}

	if len(candidates) == 1 {
		if strings.TrimSpace(candidates[0].URL) == "" {
			return nil, fmt.Errorf("asset %q is missing API url", strings.TrimSpace(candidates[0].Name))
		}
		return candidates[0], nil
	}

	if len(candidates) > 1 {
		return nil, fmt.Errorf("multiple assets match prefix/suffix for tag %s; configured=%q", release.TagName, cfg)
	}

	return nil, fmt.Errorf("asset not found in release %s (configured=%q, derived prefix=%q suffix=%q)", release.TagName, cfg, prefix, suffix)
}
