package publisher

import (
	"sync"

	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/publisher/beat"
)

type asyncClient struct {
	done <-chan struct{}

	client beat.Client
	acker  *asyncACKer

	guaranteedClient beat.Client
	guaranteedAcker  *asyncACKer
}

type asyncACKer struct {
	// Note: mutex is required for sending the message item to the
	//        asyncACKer and publishing all events, so no two users of the async client can
	//        interleave events. This is a limitation enforced by the
	//        old publisher API to be removed
	// Note: normally every go-routine wanting to publish should have it's own
	//       client instance. That is, no contention on the mutex is really expected.
	//       Still, the mutex is used as additional safety measure
	count int

	mutex   sync.Mutex
	waiting []message
}

func newAsyncClient(pub *BeatPublisher, done <-chan struct{}) (*asyncClient, error) {
	c := &asyncClient{
		done:            done,
		acker:           newAsyncACKer(),
		guaranteedAcker: newAsyncACKer(),
	}

	var err error
	c.guaranteedClient, err = pub.pipeline.ConnectWith(beat.ClientConfig{
		PublishMode: beat.GuaranteedSend,
		ACKCount:    c.guaranteedAcker.onACK,
	})
	if err != nil {
		return nil, err
	}

	c.client, err = pub.pipeline.ConnectWith(beat.ClientConfig{
		ACKCount: c.acker.onACK,
	})
	if err != nil {
		c.guaranteedClient.Close()
		return nil, err
	}

	go func() {
		// closer
		<-done
		c.guaranteedClient.Close()
		c.client.Close()
	}()

	return c, nil
}

func newAsyncACKer() *asyncACKer {
	return &asyncACKer{}
}

func (c *asyncClient) publish(m message) bool {
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
	acker := c.acker
	if m.context.Guaranteed {
		client = c.guaranteedClient
		acker = c.guaranteedAcker
	}

	acker.add(m)
	if single {
		client.Publish(m.datum)
	} else {
		client.PublishAll(m.data)
	}

	return true
}

func (a *asyncACKer) add(msg message) {
	a.mutex.Lock()
	a.waiting = append(a.waiting, msg)
	a.mutex.Unlock()
}

func (a *asyncACKer) onACK(count int) {
	for count > 0 {
		cnt := a.count
		if cnt == 0 {
			// we're not waiting for a message its ACK yet -> advance to next message
			// object and retry
			a.mutex.Lock()
			if len(a.waiting) == 0 {
				a.mutex.Unlock()
				return
			}

			active := a.waiting[0]
			cnt = len(active.data)
			a.mutex.Unlock()

			if cnt == 0 {
				cnt = 1
			}
			a.count = cnt
			continue
		}

		acked := count
		if acked > cnt {
			acked = cnt
		}
		cnt -= acked
		count -= acked

		a.count = cnt
		finished := cnt == 0
		if finished {
			var msg message

			a.mutex.Lock()
			// finished active message
			msg = a.waiting[0]
			a.waiting = a.waiting[1:]
			a.mutex.Unlock()

			if sig := msg.context.Signal; sig != nil {
				sig.Completed()
			}
		}
	}
}
