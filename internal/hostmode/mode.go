package hostmode

import "strings"

const (
	Localhost = "localhost"
	LAN       = "lan"
)

func Normalize(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case LAN:
		return LAN
	default:
		return Localhost
	}
}

func IsValid(mode string) bool {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	return normalized == "" || normalized == Localhost || normalized == LAN
}
