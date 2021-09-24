// +build !windows

package saiyan

import (
	"os"
	"os/exec"
	"syscall"
)

func bindCmd(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}
	} else {
		cmd.SysProcAttr.Setpgid = true
	}
}

func (w *work) close() {
	if w != nil {
		_ = w.Connect.Close()
		if w.Cmd.Process != nil {
			p, e := os.FindProcess(-w.Cmd.Process.Pid)
			if e == nil {
				_ = p.Signal(syscall.SIGINT)
			}
		}
		_ = w.Cmd.Process.Signal(os.Kill)
	}
}
