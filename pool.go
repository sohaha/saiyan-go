package saiyan

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/sohaha/zlsgo/znet"
)

type saiyanVar struct {
	request  *Request
	response *znet.PrevData
}

var saiyan = sync.Pool{New: func() interface{} {
	return &saiyanVar{
		request:  new(Request),
		response: new(znet.PrevData),
	}
}}

func (e *Engine) pubPool(w *work) {
	pub := true
	if e.conf.MaxRequests > 0 {
		if w.MaxRequests >= e.conf.MaxRequests {
			pub = false
		} else {
			w.MaxRequests++
		}
	}
	if pub {
		e.pool <- w
	} else {
		go e.closePool(w)
	}
}

func (e *Engine) getPool() (*work, error) {
	check := func(w *work) (*work, error) {
		if w == nil {
			return nil, ErrWorkerClose
		}
		if w.Close {
			go e.closePool(w)
			return e.getPool()
		}
		return w, nil
	}
	select {
	case w := <-e.pool:
		return check(w)
	default:
		e.mutex.Lock()
		alive := e.collectErr.aliveWorkerSum
		switch {
		case alive >= e.conf.MaxWorkerSum:
			e.mutex.Unlock()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(e.conf.MaxWaitTimeout))
			defer cancel()
			select {
			case w := <-e.pool:
				return check(w)
			case <-ctx.Done():
				return nil, ErrWorkerBusy
			}
		case alive < e.conf.MaxWorkerSum:
			e.collectErr.aliveWorkerSum++
			e.mutex.Unlock()
			return e.newWorker(true)
		}
	}
	return nil, errors.New("failed to initialize worker")
}
