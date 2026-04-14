package checkers

import "m2apps/internal/requirements"

type DartChecker struct{}

func init() {
	requirements.Register("dart", DartChecker{})
}

func (DartChecker) Check(versionConstraint string) (requirements.Result, error) {
	return checkTool("Dart", "dart", []string{"--version"}, versionConstraint)
}
