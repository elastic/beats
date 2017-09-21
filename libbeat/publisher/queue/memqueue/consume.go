package memqueue

import (
	"errors"
	"io"

	"github.com/elastic/beats/libbeat/common/atomic"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/libbeat/publisher/queue"
)

type consumer struct {
	broker *Broker
	resp   chan getResponse

	stats consumerStats

	done   chan struct{}
	closed atomic.Bool
}

type consumerStats struct {
	totalGet, totalACK uint64
}

type batch struct {
	consumer     *consumer
	events       []publisher.Event
	clientStates []clientState
	ack          *ackChan
	state        ackState
}

type ackState uint8

const (
	batchActive ackState = iota
	batchACK
)

func newConsumer(b *Broker) *consumer {
	return &consumer{
		broker: b,
		resp:   make(chan getResponse),
		done:   make(chan struct{}),
	}
}

func (c *consumer) Get(sz int) (queue.Batch, error) {
	// log := c.broker.logger

	if c.closed.Load() {
		return nil, io.EOF
	}

	select {
	case c.broker.requests <- getRequest{sz: sz, resp: c.resp}:
	case <-c.done:
		return nil, io.EOF
	}

	// if request has been send, we do have to wait for a reponse
	resp := <-c.resp

	ack := resp.ack
	c.stats.totalGet += uint64(ack.count)

	// log.Debugf("create batch: seq=%v, start=%v, len=%v", ack.seq, ack.start, len(resp.buf))
	// log.Debug("consumer: total events get = ", c.stats.totalGet)

	return &batch{
		consumer: c,
		events:   resp.buf,
		ack:      resp.ack,
		state:    batchActive,
	}, nil
}

func (c *consumer) Close() error {
	if c.closed.Swap(true) {
		return errors.New("already closed")
	}

	close(c.done)
	return nil
}

func (b *batch) Events() []publisher.Event {
	if b.state != batchActive {
		panic("Get Events from inactive batch")
	}
	return b.events
}

func (b *batch) ACK() {
	c := b.consumer
	// broker := c.broker
	// log := broker.logger

	if b.state != batchActive {
		switch b.state {
		case batchACK:
			panic("Can not acknowledge already acknowledged batch")
		default:
			panic("inactive batch")
		}
	}

	c.stats.totalACK += uint64(b.ack.count)
	// log.Debug("consumer: total events ack = ", c.stats.totalACK)
	// log.Debugf("ack batch: seq=%v, len=%v", b.ack.seq, len(b.events))
	b.report()
}

func (b *batch) report() {
	b.ack.ch <- batchAckMsg{}
}
