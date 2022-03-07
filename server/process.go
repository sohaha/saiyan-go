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
	pid := servePID()
	proc, err := os.FindProcess(pid)
	if err != nil {
		return
	}

	if zutil.IsWin() {
		err = proc.Kill()
	} else {
		err = proc.Signal(syscall.SIGQUIT)
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
	_, err := os.FindProcess(pid)
	return err == nil
}

func stopProcess() {
	serve, master := PIDFile()
	zfile.Rmdir(serve)
	zfile.Rmdir(master)
}
