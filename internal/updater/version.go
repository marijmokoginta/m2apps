package updater

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type semVersion struct {
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

func IsNewer(candidateTag, currentTag string) (bool, error) {
	cmp, err := CompareTags(candidateTag, currentTag)
	if err != nil {
		return false, err
	}
	return cmp > 0, nil
}

func CompareTags(leftTag, rightTag string) (int, error) {
	left, err := parseSemVersion(leftTag)
	if err != nil {
		return 0, fmt.Errorf("invalid version %q: %w", leftTag, err)
	}

	right, err := parseSemVersion(rightTag)
	if err != nil {
		return 0, fmt.Errorf("invalid version %q: %w", rightTag, err)
	}

	return compareSemVersion(left, right), nil
}

func parseSemVersion(tag string) (semVersion, error) {
	raw := strings.TrimSpace(tag)
	matches := semverPattern.FindStringSubmatch(raw)
	if matches == nil {
		return semVersion{}, fmt.Errorf("expected semantic version format")
	}

	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return semVersion{}, err
	}
	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return semVersion{}, err
	}

	patch := 0
	if matches[3] != "" {
		patch, err = strconv.Atoi(matches[3])
		if err != nil {
			return semVersion{}, err
		}
	}

	v := semVersion{
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

func compareSemVersion(left, right semVersion) int {
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
