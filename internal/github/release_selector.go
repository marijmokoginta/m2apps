package github

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type parsedVersion struct {
	Major     int
	Minor     int
	Patch     int
	HasPre    bool
	PreLabel  string
	PreRank   int
	PreNumber int
	PreRaw    string
}

var (
	semverPattern = regexp.MustCompile(`^v?(\d+)\.(\d+)(?:\.(\d+))?(?:-([0-9A-Za-z.-]+))?$`)
	prePattern    = regexp.MustCompile(`^([A-Za-z]+)(?:[.-]?(\d+))?$`)
)

func NormalizeChannel(channel string) string {
	ch := strings.ToLower(strings.TrimSpace(channel))
	if ch == "" {
		return "stable"
	}
	return ch
}

func MatchChannel(r Release, channel string) bool {
	ch := NormalizeChannel(channel)
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

func FilterReleasesByChannel(releases []Release, channel string) []Release {
	ch := NormalizeChannel(channel)
	var filtered []Release

	for _, release := range releases {
		if MatchChannel(release, ch) {
			filtered = append(filtered, release)
		}
	}

	return filtered
}

func GetLatestVersion(releases []Release) (*Release, error) {
	if len(releases) == 0 {
		return nil, fmt.Errorf("No releases available for channel")
	}

	valid := make([]Release, 0, len(releases))
	for _, release := range releases {
		if _, err := parseVersion(release.TagName); err == nil {
			valid = append(valid, release)
		}
	}

	if len(valid) == 0 {
		return nil, fmt.Errorf("Invalid version format")
	}

	sort.Slice(valid, func(i, j int) bool {
		cmp, _ := CompareVersionTags(valid[i].TagName, valid[j].TagName)
		return cmp > 0
	})

	selected := valid[0]
	return &selected, nil
}

func SelectLatestReleaseByChannel(client Client, owner, repo, channel string) (*Release, error) {
	releases, err := client.GetAllReleases(owner, repo)
	if err != nil {
		return nil, err
	}

	filtered := FilterReleasesByChannel(releases, channel)
	if len(filtered) == 0 {
		return nil, fmt.Errorf("No releases available for channel")
	}

	return GetLatestVersion(filtered)
}

func CompareVersionTags(leftTag, rightTag string) (int, error) {
	left, err := parseVersion(leftTag)
	if err != nil {
		return 0, fmt.Errorf("invalid version %q: %w", leftTag, err)
	}

	right, err := parseVersion(rightTag)
	if err != nil {
		return 0, fmt.Errorf("invalid version %q: %w", rightTag, err)
	}

	return compareParsedVersion(left, right), nil
}

func parseVersion(tag string) (parsedVersion, error) {
	raw := strings.TrimSpace(tag)
	matches := semverPattern.FindStringSubmatch(raw)
	if matches == nil {
		return parsedVersion{}, fmt.Errorf("expected semantic version format")
	}

	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return parsedVersion{}, err
	}
	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return parsedVersion{}, err
	}

	patch := 0
	if matches[3] != "" {
		patch, err = strconv.Atoi(matches[3])
		if err != nil {
			return parsedVersion{}, err
		}
	}

	v := parsedVersion{
		Major: major,
		Minor: minor,
		Patch: patch,
	}

	if matches[4] != "" {
		v.HasPre = true
		v.PreRaw = strings.ToLower(matches[4])
		v.PreLabel, v.PreNumber = parsePreRelease(v.PreRaw)
		v.PreRank = preLabelRank(v.PreLabel)
	}

	return v, nil
}

func compareParsedVersion(left, right parsedVersion) int {
	if left.Major != right.Major {
		return compareInt(left.Major, right.Major)
	}
	if left.Minor != right.Minor {
		return compareInt(left.Minor, right.Minor)
	}
	if left.Patch != right.Patch {
		return compareInt(left.Patch, right.Patch)
	}

	if !left.HasPre && !right.HasPre {
		return 0
	}
	if !left.HasPre && right.HasPre {
		return 1
	}
	if left.HasPre && !right.HasPre {
		return -1
	}

	if left.PreRank != right.PreRank {
		return compareInt(left.PreRank, right.PreRank)
	}

	if left.PreLabel != right.PreLabel {
		if left.PreLabel > right.PreLabel {
			return 1
		}
		if left.PreLabel < right.PreLabel {
			return -1
		}
	}

	if left.PreNumber != right.PreNumber {
		return compareInt(left.PreNumber, right.PreNumber)
	}

	if left.PreRaw > right.PreRaw {
		return 1
	}
	if left.PreRaw < right.PreRaw {
		return -1
	}

	return 0
}

func parsePreRelease(pre string) (label string, number int) {
	token := strings.Split(pre, ".")[0]
	matches := prePattern.FindStringSubmatch(token)
	if matches == nil {
		return token, 0
	}

	label = strings.ToLower(matches[1])
	if matches[2] == "" {
		return label, 0
	}

	num, err := strconv.Atoi(matches[2])
	if err != nil {
		return label, 0
	}
	return label, num
}

func preLabelRank(label string) int {
	switch strings.ToLower(label) {
	case "alpha":
		return 1
	case "beta":
		return 2
	case "rc":
		return 3
	default:
		return 0
	}
}

func compareInt(left, right int) int {
	if left > right {
		return 1
	}
	if left < right {
		return -1
	}
	return 0
}
