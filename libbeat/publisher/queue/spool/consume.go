package spool

import (
	"errors"
	"io"

	"github.com/elastic/beats/libbeat/common/atomic"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/libbeat/publisher/queue"
)

type consumer struct {
	ctx    *spoolCtx
	closed atomic.Bool
	done   chan struct{}

	resp chan getResponse
	requ chan getRequest
}

type batch struct {
	events []publisher.Event
	state  ackState
	ack    chan batchAckMsg
}

type ackState uint8

const (
	batchActive ackState = iota
	batchACK
)

func newConsumer(ctx *spoolCtx, requ chan getRequest) *consumer {
	return &consumer{
		ctx:    ctx,
		closed: atomic.MakeBool(false),
		done:   make(chan struct{}),

		// internal API
		resp: make(chan getResponse),
		requ: requ,
	}
}

func (c *consumer) Close() error {
	if c.closed.Swap(true) {
		return errors.New("already closed")
	}

	close(c.done)
	return nil
}

func (c *consumer) Closed() bool {
	return c.closed.Load() || c.ctx.Closed()
}

func (c *consumer) Get(sz int) (queue.Batch, error) {
	log := c.ctx.logger

	if c.Closed() {
		return nil, io.EOF
	}

	var resp getResponse
	for {
		select {
		case <-c.ctx.Done():
			return nil, io.EOF

		case <-c.done:
			return nil, io.EOF

		case c.requ <- getRequest{sz: sz, resp: c.resp}:
		}

		resp = <-c.resp
		err := resp.err
		if err == nil {
			break
		}

		if err != errRetry {
			log.Debug("consumer: error response:", err)
			return nil, err
		}
	}

	log.Debug("consumer: received batch:", len(resp.buf))
	return &batch{
		events: resp.buf,
		state:  batchActive,
		ack:    resp.ack,
	}, nil
}

func (b *batch) Events() []publisher.Event {
	if b.state != batchActive {
		panic("Get Events from inactive batch")
	}
	return b.events
}

func (b *batch) ACK() {
	if b.state != batchActive {
		switch b.state {
		case batchACK:
			panic("Can not acknowledge already acknowledged batch")
		default:
			panic("inactive batch")
		}
	}

	b.report()
}

func (b *batch) report() {
	if b.ack != nil {
		b.ack <- batchAckMsg{}
	}
}
