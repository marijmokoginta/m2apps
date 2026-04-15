//go:build !windows

package service

import "os"

func isRootUser() bool {
	return os.Geteuid() == 0
}
