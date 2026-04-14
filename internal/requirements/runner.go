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

		results = append(results, res)
	}

	return results
}

func normalizeName(name string) string {
	n := strings.TrimSpace(name)
	if n == "" {
		return "UNKNOWN"
	}
	return strings.ToUpper(n)
}
