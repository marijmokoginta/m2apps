//go:build windows

package selfupdate

import (
	"os/exec"
	"syscall"
)

func configureUpdaterProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
