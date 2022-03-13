package main

import (
	"os"
	"syscall"

	"github.com/sohaha/zlsgo/zfile"
	"github.com/sohaha/zlsgo/zutil"

	"github.com/sohaha/zlsgo/zcli"
)

type StartCli struct{}

func (cmd *StartCli) Flags(_ *zcli.Subcommand) {
	initDefaultFlags()
	initRunFlags()
}

func (cmd *StartCli) Run(_ []string) {
	runProcess()
}

type StopCli struct{}

func (cmd *StopCli) Flags(_ *zcli.Subcommand) {
	initDefaultFlags()
}

func (cmd *StopCli) Run(_ []string) {
	stop(servePID(), 0)
}

func stop(pid int, signal syscall.Signal) {
	proc, err := os.FindProcess(pid)
	if err != nil || proc.Pid == 0 {
		return
	}

	if signal > 0 {
		err = proc.Signal(signal)
	} else {
		if zutil.IsWin() {
			err = proc.Kill()
		} else {
			err = proc.Signal(syscall.SIGQUIT)
		}
	}

	if err != nil {
		zcli.Error(err.Error())
		return
	}

	stopProcess()
}

type RestartCli struct{}

func (cmd *RestartCli) Flags(_ *zcli.Subcommand) {
	initDefaultFlags()
}

func (cmd *RestartCli) Run(_ []string) {
	pid := masterPID()
	proc, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	_ = proc.Kill()
}

func existProcess() bool {
	pid := servePID()
	if pid == 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

func stopProcess() {
	serve, master := PIDFile()
	zfile.Rmdir(serve)
	zfile.Rmdir(master)
}
