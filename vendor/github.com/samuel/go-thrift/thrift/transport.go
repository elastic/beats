package thrift

import (
	"bufio"
	"io"
)

type Transport interface {
	ProtocolReader
	ProtocolWriter
	io.Closer
	Flusher
}

type transport struct {
	ProtocolReader
	ProtocolWriter
	io.Closer
	f Flusher
}

func NewTransport(rwc io.ReadWriteCloser, p ProtocolBuilder) Transport {
	t := &transport{
		Closer: rwc,
	}
	if _, ok := rwc.(*FramedReadWriteCloser); ok {
		t.ProtocolReader = p.NewProtocolReader(rwc)
		t.ProtocolWriter = p.NewProtocolWriter(rwc)
		if f, ok := rwc.(Flusher); ok {
			t.f = f
		}
	} else {
		w := bufio.NewWriter(rwc)
		t.ProtocolWriter = p.NewProtocolWriter(w)
		t.ProtocolReader = p.NewProtocolReader(bufio.NewReader(rwc))
		t.f = w
	}
	return t
}

func (t *transport) Flush() error {
	if t.f != nil {
		return t.f.Flush()
	}
	return nil
}
