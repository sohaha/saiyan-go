package saiyan

import (
	"github.com/sohaha/zlsgo/ztime/cron"
	"os/exec"
	"strings"
)

func (e *Engine) startTasks() error {
	if len(e.conf.CronTasks) > 0 {
		taskRun(e)
		tasker := cron.New()
		_, err := tasker.Add("* * * * *", func() {
			taskRun(e)
		})
		if err != nil {
			return err
		}
		tasker.Run()
	}
	return nil
}

func taskRun(e *Engine) {
	tasks := e.conf.CronTasks
	for i := range tasks {
		cmd := exec.Command(e.phpPath, strings.Split(tasks[i], " ")...)
		bindCmd(cmd)
		cmd.Env = e.conf.Env
		_ = cmd.Start()
	}

}
