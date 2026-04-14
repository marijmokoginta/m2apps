package requirements

import "strings"

var registry = map[string]Checker{}

func Register(name string, checker Checker) {
	registry[strings.ToLower(strings.TrimSpace(name))] = checker
}

func getChecker(name string) (Checker, bool) {
	checker, ok := registry[strings.ToLower(strings.TrimSpace(name))]
	return checker, ok
}
