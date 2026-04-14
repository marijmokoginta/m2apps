package checkers

import "m2apps/internal/requirements"

type FlutterChecker struct{}

func init() {
	requirements.Register("flutter", FlutterChecker{})
}

func (FlutterChecker) Check(versionConstraint string) (requirements.Result, error) {
	return checkTool("Flutter", "flutter", []string{"--version"}, versionConstraint)
}
