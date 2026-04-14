package checkers

import "m2apps/internal/requirements"

type MySQLChecker struct{}

func init() {
	requirements.Register("mysql", MySQLChecker{})
}

func (MySQLChecker) Check(versionConstraint string) (requirements.Result, error) {
	return checkTool("MySQL", "mysql", []string{"--version"}, versionConstraint)
}
