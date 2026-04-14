package requirements

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Version struct {
	Major int
	Minor int
	Patch int
}

var versionPattern = regexp.MustCompile(`\d+(?:\.\d+){0,2}`)

func ParseVersion(input string) (Version, error) {
	match := versionPattern.FindString(input)
	if match == "" {
		return Version{}, fmt.Errorf("invalid version output")
	}

	parts := strings.Split(match, ".")

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version: %w", err)
	}

	minor := 0
	if len(parts) > 1 {
		minor, err = strconv.Atoi(parts[1])
		if err != nil {
			return Version{}, fmt.Errorf("invalid minor version: %w", err)
		}
	}

	patch := 0
	if len(parts) > 2 {
		patch, err = strconv.Atoi(parts[2])
		if err != nil {
			return Version{}, fmt.Errorf("invalid patch version: %w", err)
		}
	}

	return Version{
		Major: major,
		Minor: minor,
		Patch: patch,
	}, nil
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func Compare(v1, v2 Version) int {
	if v1.Major != v2.Major {
		if v1.Major > v2.Major {
			return 1
		}
		return -1
	}

	if v1.Minor != v2.Minor {
		if v1.Minor > v2.Minor {
			return 1
		}
		return -1
	}

	if v1.Patch != v2.Patch {
		if v1.Patch > v2.Patch {
			return 1
		}
		return -1
	}

	return 0
}

func Satisfies(found Version, constraint string) (bool, error) {
	raw := strings.TrimSpace(constraint)
	if raw == "" {
		return false, fmt.Errorf("version constraint is required")
	}

	if !strings.HasPrefix(raw, ">=") {
		return false, fmt.Errorf("unsupported version operator")
	}

	requiredRaw := strings.TrimSpace(strings.TrimPrefix(raw, ">="))
	required, err := ParseVersion(requiredRaw)
	if err != nil {
		return false, fmt.Errorf("invalid required version")
	}

	return Compare(found, required) >= 0, nil
}
