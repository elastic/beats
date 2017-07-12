package membroker

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher/broker"
)

type Broker struct {
	done chan struct{}

	logger logger

	buf         brokerBuffer
	minEvents   int
	idleTimeout time.Duration

	// api channels
	events    chan pushRequest
	requests  chan getRequest
	pubCancel chan producerCancelRequest

	// internal channels
	acks          chan int
	scheduledACKs chan chanList

	ackSeq uint

	eventer broker.Eventer

	// wait group for worker shutdown
	wg          sync.WaitGroup
	waitOnClose bool
}

type Settings struct {
	Eventer        broker.Eventer
	Events         int
	FlushMinEvents int
	FlushTimeout   time.Duration
	WaitOnClose    bool
}

type ackChan struct {
	next         *ackChan
	ch           chan batchAckRequest
	seq          uint
	start, count int // number of events waiting for ACK
}

type chanList struct {
	head *ackChan
	tail *ackChan
}

func init() {
	broker.RegisterType("mem", create)
}

func create(eventer broker.Eventer, cfg *common.Config) (broker.Broker, error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	return NewBroker(Settings{
		Eventer:        eventer,
		Events:         config.Events,
		FlushMinEvents: config.FlushMinEvents,
		FlushTimeout:   config.FlushTimeout,
	}), nil
}

// NewBroker creates a new in-memory broker holding up to sz number of events.
// If waitOnClose is set to true, the broker will block on Close, until all internal
// workers handling incoming messages and ACKs have been shut down.
func NewBroker(
	settings Settings,
) *Broker {
	// define internal channel size for procuder/client requests
	// to the broker
	chanSize := 20

	var (
		sz           = settings.Events
		minEvents    = settings.FlushMinEvents
		flushTimeout = settings.FlushTimeout
	)

	if minEvents < 1 {
		minEvents = 1
	}
	if minEvents > 1 && flushTimeout <= 0 {
		minEvents = 1
		flushTimeout = 0
	}
	if minEvents > sz {
		minEvents = sz
	}

	logger := defaultLogger
	b := &Broker{
		done:   make(chan struct{}),
		logger: logger,

		// broker API channels
		events:    make(chan pushRequest, chanSize),
		requests:  make(chan getRequest),
		pubCancel: make(chan producerCancelRequest, 5),

		// internal broker and ACK handler channels
		acks:          make(chan int),
		scheduledACKs: make(chan chanList),

		waitOnClose: settings.WaitOnClose,

		eventer: settings.Eventer,
	}
	b.buf.init(logger, sz)
	b.minEvents = minEvents
	b.idleTimeout = flushTimeout

	ack := &ackLoop{broker: b}

	b.wg.Add(2)
	go func() {
		defer b.wg.Done()
		b.eventLoop()
	}()
	go func() {
		defer b.wg.Done()
		ack.run()
	}()

	return b
}

func (b *Broker) Close() error {
	close(b.done)
	if b.waitOnClose {
		b.wg.Wait()
	}
	return nil
}

func (b *Broker) BufferConfig() broker.BufferConfig {
	return broker.BufferConfig{
		Events: b.buf.Size(),
	}
}

func (b *Broker) Producer(cfg broker.ProducerConfig) broker.Producer {
	return newProducer(b, cfg.ACK, cfg.OnDrop, cfg.DropOnCancel)
}

func (b *Broker) Consumer() broker.Consumer {
	return newConsumer(b)
}

func (b *Broker) eventLoop() {
	var (
		timer      *time.Timer
		idleC      <-chan time.Time
		forceFlush bool

		events = b.events
		get    chan getRequest

		activeEvents int

		totalGet   uint64
		totalACK   uint64
		batchesGen uint64

		// log = b.logger

		// Buffer and send pending batches to ackloop.
		pendingACKs chanList
		schedACKS   chan chanList
	)

	if b.idleTimeout > 0 {
		// create initialy 'stopped' timer -> reset will be used
		// on timer object, if flush timer becomes active.
		timer = time.NewTimer(b.idleTimeout)
		if !timer.Stop() {
			<-timer.C
		}
	}

	for {
		select {
		case <-b.done:
			return

		// receiving new events into the buffer
		case req := <-events:
			// log.Debugf("push event: %v\t%v\t%p\n", req.event, req.seq, req.state)

			avail, ok := b.insert(req)
			if !ok {
				break
			}
			if avail == 0 {
				// log.Debugf("buffer: all regions full")
				events = nil
			}

		case req := <-b.pubCancel:
			// log.Debug("handle cancel request")
			var removed int
			if st := req.state; st != nil {
				st.cancelled = true
				removed = b.buf.cancel(st)
			}

			// signal cancel request being finished
			if req.resp != nil {
				req.resp <- producerCancelResponse{
					removed: removed,
				}
			}

			// re-enable pushRequest if buffer can take new events
			if !b.buf.Full() {
				events = b.events
			}

		case <-idleC:
			forceFlush = true
			idleC = nil

		case req := <-get:
			start, buf := b.buf.reserve(req.sz)
			count := len(buf)
			if count == 0 {
				panic("empty batch returned")
			}

			// log.Debug("newACKChan: ", b.ackSeq, count)
			ackCH := newACKChan(b.ackSeq, start, count)
			b.ackSeq++

			activeEvents += ackCH.count
			totalGet += uint64(ackCH.count)
			batchesGen++
			// log.Debug("broker: total events get = ", totalGet)
			// log.Debug("broker: total batches generated = ", batchesGen)

			req.resp <- getResponse{buf, ackCH}
			pendingACKs.append(ackCH)
			schedACKS = b.scheduledACKs

			// stop flush timer on get
			forceFlush = false
			if idleC != nil {
				idleC = nil
				if !timer.Stop() {
					<-timer.C
				}
			}

		case schedACKS <- pendingACKs:
			schedACKS = nil
			pendingACKs = chanList{}

		case count := <-b.acks:
			// log.Debug("receive buffer ack:", count)

			activeEvents -= count
			totalACK += uint64(count)
			// log.Debug("broker: total events ack = ", totalACK)

			b.buf.ack(count)
			// after ACK some buffer has been freed up, reenable publisher
			events = b.events
		}

		// update get and idle timer after state machine

		get = b.requests
		if !forceFlush {
			avail := b.buf.Avail()
			if avail == 0 || b.buf.TotalAvail() < b.minEvents {
				get = nil

				if avail > 0 && idleC == nil && timer != nil {
					timer.Reset(b.idleTimeout)
					idleC = timer.C
				}
			}
		}
	}
}

func (b *Broker) insert(req pushRequest) (int, bool) {
	var avail int
	if req.state == nil {
		_, avail = b.buf.insert(req.event, clientState{})
	} else {
		st := req.state
		if st.cancelled {
			b.logger.Debugf("cancelled producer - ignore event: %v\t%v\t%p", req.event, req.seq, req.state)

			// do not add waiting events if producer did send cancel signal

			if cb := st.dropCB; cb != nil {
				cb(req.event.Content)
			}

			return -1, false
		}

		_, avail = b.buf.insert(req.event, clientState{
			seq:   req.seq,
			state: st,
		})
	}

	return avail, true
}

func (b *Broker) reportACK(states []clientState, start, N int) {
	{
		start := time.Now()
		b.logger.Debug("handle ACKs: ", N)
		defer func() {
			b.logger.Debug("handle ACK took: ", time.Since(start))
		}()
	}

	if e := b.eventer; e != nil {
		e.OnACK(N)
	}

	// TODO: global boolean to check if clients will need an ACK
	//       no need to report ACKs if no client is interested in ACKs

	idx := start + N - 1
	if idx >= len(states) {
		idx -= len(states)
	}

	total := 0
	for i := N - 1; i >= 0; i-- {
		if idx < 0 {
			idx = len(states) - 1
		}

		st := &states[idx]
		b.logger.Debugf("try ack index: (idx=%v, i=%v, seq=%v)\n", idx, i, st.seq)

		idx--
		if st.state == nil {
			b.logger.Debug("no state set")
			continue
		}

		count := (st.seq - st.state.lastACK)
		if count == 0 || count > math.MaxUint32/2 {
			// seq number comparison did underflow. This happens only if st.seq has
			// allready been acknowledged
			b.logger.Debug("seq number already acked: ", st.seq)

			st.state = nil
			continue
		}

		b.logger.Debugf("broker ACK events: count=%v, start-seq=%v, end-seq=%v\n",
			count,
			st.state.lastACK+1,
			st.seq,
		)

		total += int(count)
		if total > N {
			panic(fmt.Sprintf("Too many events acked (expected=%v, total=%v)",
				count, total,
			))
		}

		st.state.cb(int(count))
		st.state.lastACK = st.seq
		st.state = nil
	}
}

var ackChanPool = sync.Pool{
	New: func() interface{} {
		return &ackChan{
			ch: make(chan batchAckRequest, 1),
		}
	},
}

func newACKChan(seq uint, start, count int) *ackChan {
	ch := ackChanPool.Get().(*ackChan)
	ch.next = nil
	ch.seq = seq
	ch.start = start
	ch.count = count
	return ch
}

func releaseACKChan(c *ackChan) {
	c.next = nil
	ackChanPool.Put(c)
}

func (l *chanList) prepend(ch *ackChan) {
	ch.next = l.head
	l.head = ch
	if l.tail == nil {
		l.tail = ch
	}
}

func (l *chanList) concat(other *chanList) {
	if l.head == nil {
		*l = *other
		return
	}

	l.tail.next = other.head
	l.tail = other.tail
}

func (l *chanList) append(ch *ackChan) {
	if l.head == nil {
		l.head = ch
	} else {
		l.tail.next = ch
	}
	l.tail = ch
}

func (l *chanList) count() (elems, count int) {
	for ch := l.head; ch != nil; ch = ch.next {
		elems++
		count += ch.count
	}
	return
}

func (l *chanList) empty() bool {
	return l.head == nil
}

func (l *chanList) front() *ackChan {
	return l.head
}

func (l *chanList) channel() chan batchAckRequest {
	if l.head == nil {
		return nil
	}
	return l.head.ch
}

func (l *chanList) pop() *ackChan {
	ch := l.head
	if ch != nil {
		l.head = ch.next
		if l.head == nil {
			l.tail = nil
		}
	}

	ch.next = nil
	return ch
}
