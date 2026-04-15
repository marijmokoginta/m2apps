//go:build windows

package service

func isRootUser() bool {
	return false
}
