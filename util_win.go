// +build windows

package saiyan

import (
	"os"
	"os/exec"
	"strconv"
)

func bindCmd(cmd *exec.Cmd) {
}

func (w *work) close() {
	if w != nil {
		_ = w.Connect.Close()
		if w.Cmd.Process != nil {
			cmd := exec.Command("TASKKILL", "/T", "/F", "/PID", strconv.Itoa(w.Cmd.Process.Pid))
			_, _ = cmd.CombinedOutput()
		}
		_ = w.Cmd.Process.Signal(os.Kill)

	}
}
