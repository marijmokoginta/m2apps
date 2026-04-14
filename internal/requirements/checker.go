package requirements

type Checker interface {
	Check(versionConstraint string) (Result, error)
}
