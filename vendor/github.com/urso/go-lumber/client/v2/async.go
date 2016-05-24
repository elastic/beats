package v2

import (
	"io"
	"net"
	"sync"
)

type AsyncClient struct {
	cl *Client

	inflight int
	ch       chan ackMessage
	wg       sync.WaitGroup
}

type ackMessage struct {
	cb  AsyncSendCallback
	seq uint32
	err error
}

type AsyncSendCallback func(seq uint32, err error)

func NewAsyncClientWith(cl *Client, inflight int) (*AsyncClient, error) {
	c := &AsyncClient{
		cl:       cl,
		inflight: inflight,
	}

	c.startACK()
	return c, nil
}

func NewAsyncClientWithConn(c net.Conn, inflight int, opts ...Option) (*AsyncClient, error) {
	cl, err := NewWithConn(c, opts...)
	if err != nil {
		return nil, err
	}
	return NewAsyncClientWith(cl, inflight)
}

func AsyncDial(address string, inflight int, opts ...Option) (*AsyncClient, error) {
	cl, err := Dial(address, opts...)
	if err != nil {
		return nil, err
	}
	return NewAsyncClientWith(cl, inflight)
}

func AsyncDialWith(
	dial func(network, address string) (net.Conn, error),
	address string,
	inflight int,
	opts ...Option,
) (*AsyncClient, error) {
	cl, err := DialWith(dial, address, opts...)
	if err != nil {
		return nil, err
	}
	return NewAsyncClientWith(cl, inflight)
}

func (c *AsyncClient) Close() error {
	err := c.cl.Close()
	c.stopACK()
	return err
}

func (c *AsyncClient) Send(cb AsyncSendCallback, data []interface{}) error {
	if err := c.cl.Send(data); err != nil {
		c.ch <- ackMessage{
			seq: 0,
			cb:  cb,
			err: err,
		}
		return err
	}

	c.ch <- ackMessage{
		seq: uint32(len(data)),
		cb:  cb,
		err: nil,
	}
	return nil
}

func (c *AsyncClient) startACK() {
	c.ch = make(chan ackMessage, c.inflight)
	c.wg.Add(1)
	go c.ackLoop()
}

func (c *AsyncClient) stopACK() {
	close(c.ch)
	c.wg.Wait()
}

func (c *AsyncClient) ackLoop() {
	var seq uint32
	var err error

	// drain ack queue on error/exit
	defer func() {
		if err == nil {
			err = io.EOF
		}
		for msg := range c.ch {
			if msg.err != nil {
				err = msg.err
			}
			msg.cb(0, err)
		}
	}()
	defer c.wg.Done()

	for msg := range c.ch {
		if msg.err != nil {
			err = msg.err
			msg.cb(msg.seq, msg.err)
			return
		}

		seq, err = c.cl.AwaitACK(msg.seq)
		msg.cb(seq, err)
		if err != nil {
			c.cl.Close()
			return
		}
	}
}
