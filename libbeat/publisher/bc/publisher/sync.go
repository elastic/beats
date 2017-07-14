package publisher

import (
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/publisher/beat"
)

type syncClient struct {
	client           beat.Client
	guaranteedClient beat.Client
	done             <-chan struct{}
	active           syncMsgContext
}

type syncMsgContext struct {
	count int
	sig   chan struct{}
}

func newSyncClient(pub *BeatPublisher, done <-chan struct{}) (*syncClient, error) {
	// always assume sync client is used with 'guarnateed' flag (true for filebeat and winlogbeat)

	c := &syncClient{done: done}
	c.active.init()

	var err error
	c.guaranteedClient, err = pub.pipeline.ConnectWith(beat.ClientConfig{
		PublishMode: beat.GuaranteedSend,
		ACKCount:    c.onACK,
	})
	if err != nil {
		return nil, err
	}

	c.client, err = pub.pipeline.ConnectWith(beat.ClientConfig{
		ACKCount: c.onACK,
	})
	if err != nil {
		c.guaranteedClient.Close()
		return nil, err
	}

	go func() {
		<-done
		c.client.Close()
		c.guaranteedClient.Close()
	}()

	return c, nil
}

func (c *syncClient) onACK(count int) {
	c.active.count -= count
	if c.active.count < 0 {
		panic("negative event count")
	}

	if c.active.count == 0 {
		c.active.sig <- struct{}{}
	}
}

func (c *syncClient) publish(m message) bool {
	if *publishDisabled {
		debug("publisher disabled")
		op.SigCompleted(m.context.Signal)
		return true
	}

	count := len(m.data)
	single := count == 0
	if single {
		count = 1
	}

	client := c.client
	if m.context.Guaranteed {
		client = c.guaranteedClient
	}

	c.active.count = count
	if single {
		client.Publish(m.datum)
	} else {
		client.PublishAll(m.data)
	}

	// wait for event or close
	select {
	case <-c.done:
		return false
	case <-c.active.sig:
	}

	if s := m.context.Signal; s != nil {
		s.Completed()
	}

	return true
}

func (ctx *syncMsgContext) init() {
	ctx.sig = make(chan struct{}, 1)
}
