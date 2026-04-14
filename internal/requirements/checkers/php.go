package checkers

import "m2apps/internal/requirements"

type PHPChecker struct{}

func init() {
	requirements.Register("php", PHPChecker{})
}

func (PHPChecker) Check(versionConstraint string) (requirements.Result, error) {
	return checkTool("PHP", "php", []string{"-v"}, versionConstraint)
}
