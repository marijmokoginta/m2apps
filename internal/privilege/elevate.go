package privilege

func IsElevated() bool {
	return isElevated()
}

func RelaunchElevated(args []string) error {
	return relaunchElevated(args)
}
