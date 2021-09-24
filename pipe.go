package saiyan

import (
	"errors"
	"io"
	"math"
)

type PipeRelay struct {
	in  io.ReadCloser
	out io.WriteCloser
}

func NewPipeRelay(in io.ReadCloser, out io.WriteCloser) *PipeRelay {
	return &PipeRelay{in: in, out: out}
}

func (pr *PipeRelay) Send(data []byte, flags byte) (err error) {
	prefix := NewPrefix().WithFlags(flags).WithSize(uint64(len(data)))
	if _, err := pr.out.Write(append(prefix[:], data...)); err != nil {
		return err
	}
	return nil
}

func (pr *PipeRelay) Receive() (data []byte, p Prefix, err error) {
	defer func() {
		if rErr, ok := recover().(error); ok {
			err = rErr
		}
	}()
	if _, err := pr.in.Read(p[:]); err != nil {
		return nil, p, err
	}
	if !p.Valid() {
		return nil, p, errors.New("invalid data found in the buffer")
	}
	if !p.HasPayload() {
		return nil, p, nil
	}
	data = make([]byte, 0, p.Size())
	leftBytes := p.Size()
	buffer := make([]byte, uint(math.Min(float64(cap(data)), float64(BufferSize))))
	for {
		if n, err := pr.in.Read(buffer); err == nil {
			data = append(data, buffer[:n]...)
			leftBytes -= uint64(n)
		} else {
			return nil, p, err
		}

		if leftBytes == 0 {
			break
		}
	}
	return
}

func (pr *PipeRelay) Close() error {
	return nil
}
