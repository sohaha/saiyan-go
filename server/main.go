package main

import (
	"os"
	"strconv"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/sohaha/zlsgo/zutil"

	"github.com/sohaha/saiyan-go"
	"github.com/sohaha/zlsgo/zcli"
	"github.com/sohaha/zlsgo/zfile"
	"github.com/sohaha/zlsgo/znet"
	"github.com/sohaha/zlsgo/zstring"
)

var (
	projectPath *string
	port        *string
	max         *int
)

func main() {
	initDefaultFlags()
	initRunFlags()

	zcli.Name = "Saiyan"
	zcli.Version = "0.1.1"
	zcli.EnableDetach = true

	zcli.Add("start", "Start Serve", &StartCli{})
	zcli.Add("stop", "Stop Serve", &StopCli{})
	zcli.Add("restart", "Restart Serve", &RestartCli{})

	zcli.Run(runProcess)

	stopProcess()
}

func PIDFile() (serve string, master string) {
	files := zutil.Once(func() interface{} {
		*projectPath = zfile.RealPath(*projectPath)
		storagePath := *projectPath + "/storage"
		serve = storagePath + "/saiyan/server.pid"
		master = storagePath + "/saiyan/master.pid"
		return []string{serve, master}
	})().([]string)
	master = files[1]
	serve = files[0]
	return
}

func getPID(file string) int {
	pid, err := zfile.ReadFile(file)
	if err != nil {
		return 0
	}

	i, err := strconv.ParseInt(zstring.TrimSpace(string(pid)), 10, 64)
	if err != nil {
		return 0
	}
	return int(i)
}

func servePID() int {
	serve, _ := PIDFile()
	return getPID(serve)
}

func masterPID() int {
	_, master := PIDFile()
	return getPID(master)
}

func runProcess() {
	if existProcess() {
		zcli.Error("unable to start multiple instances")
	}
	r := znet.New()

	// cfg := initConf()
	// if cfg.Debug {
	//	r.SetMode(znet.DebugMode)
	// }

	w, err := saiyan.New(func(conf *saiyan.Config) {
		conf.ProjectPath = *projectPath
		if *max > 0 {
			conf.MaxWorkerSum = uint64(*max)
		}
	})

	if err != nil {
		zcli.Error(err.Error())
	}

	defer w.Close()

	w.BindHttpHandler(r)

	r.SetAddr(*port)

	znet.ShutdownDone = stopProcess

	serve, master := PIDFile()
	err = zutil.TryCatch(func() error {
		pid := os.Getpid()
		zutil.CheckErr(zfile.WriteFile(serve, []byte(strconv.Itoa(pid))))
		zutil.CheckErr(zfile.WriteFile(master, []byte(strconv.Itoa(w.PID()))))
		watchPID(serve, pid)
		return nil
	})
	if err != nil {
		zcli.Error(err.Error())
	}

	znet.Run()
}

func watchPID(serve string, pid int) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op == fsnotify.Remove {
					stop(pid, syscall.SIGINT)
				}
			}
		}
	}()
	_ = watcher.Add(serve)
}
