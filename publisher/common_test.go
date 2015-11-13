package publisher

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/outputs"
)

// testMessageHandler receives messages and acknowledges them through
// their Signaler.
type testMessageHandler struct {
	msgs     chan message   // Channel that hold received messages.
	response OutputResponse // Response type to give to received messages.
	stopped  uint32         // Indicates if the messageHandler has been stopped.
}

var _ messageHandler = &testMessageHandler{}
var _ worker = &testMessageHandler{}

func (mh *testMessageHandler) onMessage(m message) {
	mh.msgs <- m
	mh.acknowledgeMessage(m)
}

func (mh *testMessageHandler) onStop() {
	atomic.AddUint32(&mh.stopped, 1)
}

func (mh *testMessageHandler) send(m message) {
	mh.msgs <- m
	mh.acknowledgeMessage(m)
}

func (mh *testMessageHandler) acknowledgeMessage(m message) {
	if mh.response == CompletedResponse {
		outputs.SignalCompleted(m.context.signal)
	} else {
		outputs.SignalFailed(m.context.signal, nil)
	}
}

// waitForMessages waits for n messages to be received and then returns. If n
// messages are not received within one second the method returns an error.
func (mh *testMessageHandler) waitForMessages(n int) ([]message, error) {
	var msgs []message
	for {
		select {
		case m := <-mh.msgs:
			msgs = append(msgs, m)
			if len(msgs) == n {
				return msgs, nil
			}
		case <-time.After(10 * time.Second):
			return nil, fmt.Errorf("Expected %d messages but received %d.",
				n, len(msgs))
		}
	}
}

type testSignaler struct {
	nonBlockingStatus chan bool // Contains status if read by isDone.
	status            chan bool // Contains Completed/Failed status.
}

func newTestSignaler() *testSignaler {
	return &testSignaler{
		status: make(chan bool, 1),
	}
}

var _ outputs.Signaler = &testSignaler{}

// Returns true if a signal was received. Never blocks.
func (s *testSignaler) isDone() bool {
	select {
	case status := <-s.status:
		s.nonBlockingStatus <- status
		return true
	default:
		return false
	}
}

// Waits for a signal to be received. Returns true if
// Completed was invoked and false if Failed was invoked.
func (s *testSignaler) wait() bool {
	select {
	case s := <-s.nonBlockingStatus:
		return s
	case s := <-s.status:
		return s
	}
}

func (s *testSignaler) Completed() {
	s.status <- true
}

func (s *testSignaler) Failed() {
	s.status <- false
}

// testEvent returns a new common.MapStr with the required fields
// populated.
func testEvent() common.MapStr {
	event := common.MapStr{}
	event["@timestamp"] = common.Time(time.Now())
	event["type"] = "test"
	event["src"] = &common.Endpoint{}
	event["dst"] = &common.Endpoint{}
	return event
}

type testPublisher struct {
	pub              *PublisherType
	outputMsgHandler *testMessageHandler
}

const (
	BulkOn  = true
	BulkOff = false
)

type OutputResponse bool

const (
	CompletedResponse OutputResponse = true
	FailedResponse    OutputResponse = false
)

func newTestPublisher(bulkSize int, response OutputResponse) *testPublisher {
	mh := &testMessageHandler{
		msgs:     make(chan message, 10),
		response: response,
	}

	ow := &outputWorker{}
	ow.config.BulkMaxSize = &bulkSize
	ow.handler = mh
	ws := workerSignal{}
	ow.messageWorker.init(&ws, 1000, mh)

	pub := &PublisherType{
		Output:   []*outputWorker{ow},
		wsOutput: ws,
	}
	pub.wsOutput.Init()
	pub.wsPublisher.Init()
	pub.syncPublisher = newSyncPublisher(pub)
	pub.asyncPublisher = newAsyncPublisher(pub)
	return &testPublisher{
		pub:              pub,
		outputMsgHandler: mh,
	}
}

func (t *testPublisher) asyncPublishEvent(event common.MapStr) bool {
	ctx := context{}
	return t.pub.asyncPublisher.client().PublishEvent(&ctx, event)
}

func (t *testPublisher) asyncPublishEvents(events []common.MapStr) bool {
	ctx := context{}
	return t.pub.asyncPublisher.client().PublishEvents(&ctx, events)
}

func (t *testPublisher) syncPublishEvent(event common.MapStr) bool {
	ctx := context{publishOptions: publishOptions{confirm: true}}
	return t.pub.syncPublisher.client().PublishEvent(&ctx, event)
}

func (t *testPublisher) syncPublishEvents(events []common.MapStr) bool {
	ctx := context{publishOptions: publishOptions{confirm: true}}
	return t.pub.syncPublisher.client().PublishEvents(&ctx, events)
}

// newTestPublisherWithBulk returns a new testPublisher with bulk message
// dispatching enabled.
func newTestPublisherWithBulk(response OutputResponse) *testPublisher {
	return newTestPublisher(defaultBulkSize, response)
}

// newTestPublisherWithBulk returns a new testPublisher with bulk message
// dispatching disabled.
func newTestPublisherNoBulk(response OutputResponse) *testPublisher {
	return newTestPublisher(-1, response)
}

func testMessage(s *testSignaler, event common.MapStr) message {
	return message{context: context{signal: s}, event: event}
}

func testBulkMessage(s *testSignaler, events []common.MapStr) message {
	return message{context: context{signal: s}, events: events}
}
