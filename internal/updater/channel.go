package updater

import (
	"strings"

	"m2apps/internal/github"
)

func MatchChannel(r github.Release, channel string) bool {
	ch := normalizeChannel(channel)
	tag := strings.ToLower(strings.TrimSpace(r.TagName))

	switch ch {
	case "stable":
		return !r.Prerelease
	case "beta":
		return r.Prerelease && strings.Contains(tag, "beta")
	case "alpha":
		return r.Prerelease && strings.Contains(tag, "alpha")
	default:
		return false
	}
}

func normalizeChannel(channel string) string {
	ch := strings.ToLower(strings.TrimSpace(channel))
	if ch == "" {
		return "stable"
	}
	return ch
}
