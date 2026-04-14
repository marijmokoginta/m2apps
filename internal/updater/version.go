package updater

import "m2apps/internal/github"

func IsNewer(candidateTag, currentTag string) (bool, error) {
	cmp, err := github.CompareVersionTags(candidateTag, currentTag)
	if err != nil {
		return false, err
	}
	return cmp > 0, nil
}

func CompareTags(leftTag, rightTag string) (int, error) {
	return github.CompareVersionTags(leftTag, rightTag)
}
