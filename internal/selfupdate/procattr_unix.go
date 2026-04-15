//go:build !windows

package selfupdate

import (
	"os/exec"
	"syscall"
)

func configureUpdaterProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
}
