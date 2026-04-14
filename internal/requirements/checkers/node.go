package checkers

import "m2apps/internal/requirements"

type NodeChecker struct{}

func init() {
	requirements.Register("node", NodeChecker{})
}

func (NodeChecker) Check(versionConstraint string) (requirements.Result, error) {
	return checkTool("Node", "node", []string{"-v"}, versionConstraint)
}
