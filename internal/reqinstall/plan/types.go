package plan

type MissingRequirement struct {
	ToolType string
	Name     string
	Required string
	Found    string
	Reason   string
}

type InstallCandidate struct {
	ToolType        string
	Name            string
	RequiredVersion string
	TargetVersion   string
	OS              string
	Method          string
	Commands        []string
	Notes           string
}

type InstallPlan struct {
	Missing    []MissingRequirement
	Candidates []InstallCandidate
	Warnings   []string
}
