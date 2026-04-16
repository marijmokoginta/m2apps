package requirements

type Result struct {
	Name     string
	Required string
	Found    string
	Success  bool
	Missing  bool
	Reason   string
	Message  string
}

const (
	ReasonNotFound        = "not_found"
	ReasonVersionMismatch = "version_mismatch"
	ReasonUnknown         = "unknown"
)
