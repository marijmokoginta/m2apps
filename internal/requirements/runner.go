package requirements

import "strings"

func Run(reqs []Requirement) []Result {
	var results []Result

	for _, r := range reqs {
		checker, ok := getChecker(r.Type)
		if !ok {
			results = append(results, Result{
				Name:     normalizeName(r.Type),
				Required: r.Version,
				Success:  false,
				Missing:  true,
				Reason:   ReasonUnknown,
				Message:  "unknown requirement type",
			})
			continue
		}

		res, err := checker.Check(r.Version)
		if strings.TrimSpace(res.Name) == "" {
			res.Name = normalizeName(r.Type)
		}
		if strings.TrimSpace(res.Required) == "" {
			res.Required = r.Version
		}

		if err != nil {
			res.Success = false
			if strings.TrimSpace(res.Message) == "" {
				res.Message = err.Error()
			}
		}

		res.Missing, res.Reason = classifyResult(res, err)

		results = append(results, res)
	}

	return results
}

func classifyResult(res Result, checkErr error) (bool, string) {
	if res.Success {
		return false, ""
	}

	if strings.EqualFold(strings.TrimSpace(res.Found), "not found") {
		return true, ReasonNotFound
	}

	errMsg := ""
	if checkErr != nil {
		errMsg = strings.ToLower(strings.TrimSpace(checkErr.Error()))
	}
	if errMsg == "not found" || strings.Contains(errMsg, "not found") {
		return true, ReasonNotFound
	}

	if strings.TrimSpace(res.Found) != "" && strings.TrimSpace(res.Required) != "" {
		return true, ReasonVersionMismatch
	}

	return true, ReasonUnknown
}

func normalizeName(name string) string {
	n := strings.TrimSpace(name)
	if n == "" {
		return "UNKNOWN"
	}
	return strings.ToUpper(n)
}
