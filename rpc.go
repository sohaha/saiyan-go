package saiyan

import (
	"errors"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"strconv"

	"github.com/sohaha/zlsgo/znet"
	"github.com/sohaha/zlsgo/zpool"
)

type RPC struct {
	addr     string
	listener net.Listener
}

// RegisterRPC Register publishes the receiver's methods
func RegisterRPC(rcvr ...interface{}) (*RPC, error) {
	if len(rcvr) == 0 {
		return nil, errors.New("receiver's methods cannot be empty")
	}
	for i := range rcvr {
		rpc.Register(rcvr[i])
	}
	prot, _ := znet.Port(0, true)
	addr := "127.0.0.1:" + strconv.Itoa(prot)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &RPC{
		addr:     addr,
		listener: lis,
	}, nil
}

func (r *RPC) String() string {
	return r.addr
}

func (r *RPC) Accept(pool uint64) {
	p := zpool.New(int(pool))
	for {
		conn, err := r.listener.Accept()
		if err != nil {
			continue
		}
		_ = p.Do(func() {
			jsonrpc.ServeConn(conn)
		})
	}
}
