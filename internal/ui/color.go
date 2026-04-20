package ui

const (
	Reset  = "\033[0m"
	Green  = "\033[32m"
	Red    = "\033[31m"
	Yellow = "\033[33m"
	Blue   = "\033[94m"
)

func Success(msg string) string {
	return Green + msg + Reset
}

func Error(msg string) string {
	return Red + msg + Reset
}

func Warning(msg string) string {
	return Yellow + msg + Reset
}

func Info(msg string) string {
	return Blue + msg + Reset
}
