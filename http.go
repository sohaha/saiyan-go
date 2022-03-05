package saiyan

import (
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/sohaha/zlsgo/zfile"
	"github.com/sohaha/zlsgo/znet"
)

func (e *Engine) BindHttpHandler(r *znet.Engine, middlewares ...znet.HandlerFunc) {
	r.Any("/", e.httpHandler, middlewares...)
	r.Any("*", e.httpHandler, middlewares...)
}

func (e *Engine) httpHandler(c *znet.Context) {
	if file, ok := e.exportFile(c.Request.URL.Path); ok {
		c.File(file)
		return
	}
	v, _ := saiyan.Get().(*saiyanVar)
	defer func() {
		saiyan.Put(v)
	}()
	err := e.newRequest(c, c.Request, v)
	if err != nil {
		e.httpErr(c, err)
		return
	}
	header, result, prefix, err := e.sendRequest(v)
	if err != nil {
		e.httpErr(c, err)
		return
	}

	e.newResponse(c, v, header, result, prefix)
}

func (e *Engine) httpErr(c *znet.Context, err error) {
	c.WithValue(HttpErrKey, err)
	if c.Engine.IsDebug() {
		c.String(500, err.Error())
	} else {
		c.Abort(500)
	}

	go func() {
		if err != nil {
			switch err {
			case ErrExecTimeout:
				atomic.AddUint64(&e.collectErr.ExecTimeout, 1)
			case ErrProcessDeath:
				atomic.AddUint64(&e.collectErr.ProcessDeath, 1)
			case ErrWorkerBusy:
				atomic.AddUint64(&e.collectErr.QueueTimeout, 1)
			default:
				atomic.AddUint64(&e.collectErr.UnknownFailed, 1)
			}
		}
	}()
}

func (e *Engine) exportFile(file string) (string, bool) {
	dirs := e.conf.StaticResourceDirs
	forbidExt := e.conf.ForbidStaticResourceSuffix
	file = file[1:]
	for i := range dirs {
		if strings.HasPrefix(file, dirs[i]) {
			file = e.conf.ProjectPath + "/public/" + file
			ext := filepath.Ext(file)
			if ext == "" {
				file = file + "index.html"
				ext = ".html"
			}
			if !zfile.FileExist(file) {
				return "", false
			}
			for i := range forbidExt {
				if ext == forbidExt[i] {
					return "", false
				}
			}
			return file, true
		}
	}
	return "", false
}
