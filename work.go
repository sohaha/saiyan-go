package saiyan

import (
	"context"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/sohaha/zlsgo/zfile"
	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/zstring"
)

type (
	Engine struct {
		conf       *Config
		phpPath    string
		pool       chan *work
		stop       chan struct{}
		restart    chan struct{}
		mutex      sync.RWMutex
		mainCmd    *exec.Cmd
		collectErr *EngineCollect
	}
	EngineCollect struct {
		ExecTimeout    uint64
		QueueTimeout   uint64
		ProcessDeath   uint64
		UnknownFailed  uint64
		aliveWorkerSum uint64
	}
	work struct {
		Connect     *PipeRelay
		Cmd         *exec.Cmd
		MaxRequests uint64
		Close       bool
	}
	Config struct {
		PHPExecPath                string
		Command                    string
		WorkerSum                  uint64
		MaxWorkerSum               uint64
		finalMaxWorkerSum          uint64
		ReleaseTime                uint64
		MaxRequests                uint64
		MaxWaitTimeout             uint64
		MaxExecTimeout             uint64
		TrimPrefix                 string
		StaticResourceDir          string
		ForbidStaticResourceSuffix []string
		Env                        []string
		CronTasks                  []string
		JSONRPC                    *RPC
	}
	Conf func(conf *Config)
)

func New(config ...Conf) (e *Engine, err error) {
	cpu := runtime.NumCPU()
	zlsPath := zfile.RealPath("zls")
	c := &Config{
		Command:                    zlsPath + "saiyan start",
		WorkerSum:                  uint64(cpu),
		MaxWorkerSum:               uint64(cpu * 2),
		ReleaseTime:                1800,
		MaxRequests:                1 << 20,
		MaxWaitTimeout:             60,
		MaxExecTimeout:             180,
		StaticResourceDir:          "public",
		Env:                        []string{},
		ForbidStaticResourceSuffix: []string{".php"},
	}
	for i := range config {
		config[i](c)
	}
	if c.WorkerSum == 0 {
		c.WorkerSum = 1
	}
	if c.MaxWorkerSum == 0 {
		c.MaxWorkerSum = c.WorkerSum / 2
	}
	c.finalMaxWorkerSum = c.MaxWorkerSum * 2
	c.StaticResourceDir = strings.TrimSuffix(c.StaticResourceDir, "/")

	// c.Env = append(c.Env, os.Environ()...)
	c.Env = append(c.Env, "SAIYAN_VERSION="+VERSUION)
	c.Env = append(c.Env, "ZLSPHP_WORKS=saiyan")

	if c.JSONRPC != nil {
		addr := c.JSONRPC.String()
		c.Env = append(c.Env, "ZLSPHP_JSONRPC_ADDR="+addr)
		go c.JSONRPC.Accept(int(c.MaxWorkerSum) * 2)
	}

	c.Command = zstring.TrimSpace(c.Command)

	e = &Engine{
		conf:       c,
		pool:       make(chan *work, c.finalMaxWorkerSum),
		collectErr: &EngineCollect{},
		stop:       make(chan struct{}),
		restart:    make(chan struct{}),
	}

	if e.phpPath, err = getPHP(c.PHPExecPath, true); err != nil {
		return
	}

	e.mainCmd, err = mainWork(e)
	if err != nil {
		return
	}

	for i := uint64(0); i < c.WorkerSum; i++ {
		e.collectErr.aliveWorkerSum++
		var w *work
		w, err = e.newWorker(true)
		if err != nil {
			return
		}
		e.pubPool(w)
	}
	err = e.startTasks()
	e.cronRelease()
	return
}

func (e *Engine) cronRelease() {
	if e.conf.ReleaseTime != 0 {
		go func() {
			t := time.NewTicker(time.Duration(e.conf.ReleaseTime) * time.Second)
			for {
				<-t.C
				if e == nil {
					t.Stop()
					return
				}
				if e.Cap() != 0 {
					e.Release(e.conf.WorkerSum)
				}
			}
		}()
	}
}

func (e *Engine) Cap() uint64 {
	e.mutex.Lock()
	aliveWorkerSum := e.collectErr.aliveWorkerSum
	e.mutex.Unlock()
	return aliveWorkerSum
}

func (e *Engine) Collect() EngineCollect {
	if e == nil {
		return EngineCollect{}
	}
	return *e.collectErr
}

func (e *Engine) IsClose() bool {
	select {
	case <-e.stop:
		return true
	default:
		return false
	}
}

func (e *Engine) Close() {
	e.stop <- struct{}{}
	if e.mainCmd != nil {
		_ = e.mainCmd.Process.Kill()
	}
	e.release(0)
}

func (e *Engine) Restart() {
	e.restart <- struct{}{}
}

func (e *Engine) release(alive uint64) {
	if alive > 0 {
		i := alive
		for {
			e.mutex.Lock()
			if e.collectErr.aliveWorkerSum == 0 || (e.collectErr.aliveWorkerSum <= e.conf.WorkerSum || i == 0) {
				e.mutex.Unlock()
				break
			}
			e.collectErr.aliveWorkerSum--
			e.mutex.Unlock()
			p := <-e.pool
			p.close()
			i--
		}
		return
	}
	e.mutex.Lock()
	for 0 < e.collectErr.aliveWorkerSum {
		e.collectErr.aliveWorkerSum--
		p := <-e.pool
		p.close()
	}
	e.mutex.Unlock()
	e.collectErr = &EngineCollect{}
}

func (e *Engine) Release(aliveWorker ...uint64) {
	alive := e.conf.WorkerSum
	if len(aliveWorker) > 0 {
		alive = aliveWorker[0]
	}
	e.mutex.Lock()
	current := e.collectErr.aliveWorkerSum
	e.mutex.Unlock()
	if current <= alive {
		return
	}
	e.release(current - alive)
}

func (e *Engine) SendNoResult(data []byte, flags byte) (err error) {
	var w *work
	w, err = e.getPool()
	if err != nil {
		return
	}
	return w.Connect.Send(data, flags)
}

func (e *Engine) sendRequest(v *saiyanVar) (headerResult, result []byte, prefix Prefix, err error) {
	var header []byte
	header, err = zjson.Marshal(v.request)
	if err != nil {
		return
	}
	var w *work
	w, err = e.getPool()
	if err != nil {
		return
	}
	defer func() {
		if err == nil {
			go e.pubPool(w)
		} else {
			go e.closePool(w)
		}
	}()
	err = w.Connect.Send(header, PayloadControl)
	if err != nil {
		return
	}
	var body []byte
	if v.request.Parsed {
		if body, err = zjson.Marshal(v.request.body); err != nil {
			return
		}
	} else if v.request.body != nil {
		var ok bool
		body, ok = v.request.body.([]byte)
		if !ok {
			if s, ok := v.request.body.(string); ok {
				body = zstring.String2Bytes(s)
			}
		}
	}
	headerResult, prefix, err = w.send(body, 0, e.conf.MaxExecTimeout)
	if err == nil {
		result, _, err = w.Connect.Receive()
	}
	if err == io.EOF {
		err = ErrProcessDeath
	}
	return
}

func (e *Engine) SendTask(taskName string) (result []byte, err error) {
	json, _ := zjson.Set(`{"type":"task"}`, "task", taskName)
	result, _, err = e.Send(zstring.String2Bytes(json), PayloadControl|PayloadRaw)
	return
}

func (e *Engine) Send(data []byte, flags byte) (result []byte, prefix Prefix, err error) {
	var w *work
	w, err = e.getPool()
	if err != nil {
		return
	}
	result, prefix, err = w.send(data, flags, e.conf.MaxExecTimeout)
	if err == nil {
		go e.pubPool(w)
	} else {
		go e.closePool(w)
	}
	if err == io.EOF {
		err = ErrProcessDeath
	}
	return
}

func (e *Engine) closePool(w *work) {
	e.mutex.Lock()
	e.collectErr.aliveWorkerSum--
	e.mutex.Unlock()
	w.close()
}

func (e *Engine) newWorker(auto bool) (*work, error) {
	if e.IsClose() {
		return nil, ErrWorkerClose
	}
	var (
		err error
		in  io.ReadCloser
		out io.WriteCloser
		cmd = exec.Command(e.phpPath, strings.Split(e.conf.Command, " ")...)
	)
	bindCmd(cmd)
	cmd.Env = e.conf.Env
	if in, err = cmd.StdoutPipe(); err != nil {
		return nil, err
	}
	if out, err = cmd.StdinPipe(); err != nil {
		return nil, err
	}
	connect := NewPipeRelay(in, out)
	if err = cmd.Start(); err != nil {
		return nil, err
	}
	w := &work{
		Cmd:     cmd,
		Connect: connect,
		Close:   false,
	}
	if auto {
		go func() {
			_ = cmd.Wait()
			if w != nil {
				w.Close = true
			}
		}()
	}
	return w, nil
}

func (w *work) send(data []byte, flags byte, maxExecTimeout uint64) (result []byte, prefix Prefix, err error) {
	err = w.Connect.Send(data, flags)
	if err != nil {
		return
	}
	ch := make(chan struct{})
	kill := make(chan bool)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(maxExecTimeout))
	defer cancel()
	go func() {
		result, prefix, err = w.Connect.Receive()
		ch <- struct{}{}
	}()
	go func() {
		if <-kill {
			w.close()
		}
	}()
	select {
	case <-ch:
		kill <- false
	case <-ctx.Done():
		err = ErrExecTimeout
		kill <- true
	}
	return
}
