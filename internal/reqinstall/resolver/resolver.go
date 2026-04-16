package resolver

import (
	"regexp"
	"strings"
)

var versionPattern = regexp.MustCompile(`\d+(?:\.\d+){0,2}`)

func Resolve(_ string, constraint string) string {
	raw := strings.TrimSpace(constraint)
	if raw == "" {
		return "stable"
	}

	matched := versionPattern.FindString(raw)
	if strings.TrimSpace(matched) == "" {
		return "stable"
	}

	return matched
}
