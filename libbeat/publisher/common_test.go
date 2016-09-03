// +build !integration

package publisher

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
)

func enableLogging(selectors []string) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, selectors)
	}
}

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
		op.SigCompleted(m.context.Signal)
	} else {
		op.SigFailed(m.context.Signal, nil)
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

var _ op.Signaler = &testSignaler{}

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

func (s *testSignaler) Canceled() {
	s.status <- true
}

// testEvent returns a new common.MapStr with the required fields
// populated.
func testEvent() outputs.Data {
	return outputs.Data{Event: common.MapStr{
		"@timestamp": common.Time(time.Now()),
		"type":       "test",
		"src":        &common.Endpoint{},
		"dst":        &common.Endpoint{},
	}}
}

type testPublisher struct {
	pub              *BeatPublisher
	outputMsgHandler *testMessageHandler
	client           *client
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
	pub := &BeatPublisher{}
	pub.wsOutput.Init()
	pub.wsPublisher.Init()

	mh := &testMessageHandler{
		msgs:     make(chan message, 10),
		response: response,
	}

	ow := &outputWorker{}
	ow.config.BulkMaxSize = bulkSize
	ow.handler = mh
	ow.messageWorker.init(&pub.wsOutput, DefaultQueueSize, DefaultBulkQueueSize, mh)

	pub.Output = []*outputWorker{ow}

	pub.pipelines.sync = newSyncPipeline(pub, DefaultQueueSize, DefaultBulkQueueSize)
	pub.pipelines.async = newAsyncPipeline(pub, DefaultQueueSize, DefaultBulkQueueSize, &pub.wsPublisher)

	return &testPublisher{
		pub:              pub,
		outputMsgHandler: mh,
		client:           pub.Connect().(*client),
	}
}

func (t *testPublisher) Stop() {
	t.client.Close()
	t.pub.Stop()
}

func (t *testPublisher) asyncPublishEvent(data outputs.Data) bool {
	ctx := Context{}
	msg := message{client: t.client, context: ctx, datum: data}
	return t.pub.pipelines.async.publish(msg)
}

func (t *testPublisher) asyncPublishEvents(data []outputs.Data) bool {
	ctx := Context{}
	msg := message{client: t.client, context: ctx, data: data}
	return t.pub.pipelines.async.publish(msg)
}

func (t *testPublisher) syncPublishEvent(data outputs.Data) bool {
	ctx := Context{publishOptions: publishOptions{Guaranteed: true}}
	msg := message{client: t.client, context: ctx, datum: data}
	return t.pub.pipelines.sync.publish(msg)
}

func (t *testPublisher) syncPublishEvents(data []outputs.Data) bool {
	ctx := Context{publishOptions: publishOptions{Guaranteed: true}}
	msg := message{client: t.client, context: ctx, data: data}
	return t.pub.pipelines.sync.publish(msg)
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

func testMessage(s *testSignaler, data outputs.Data) message {
	return message{context: Context{Signal: s}, datum: data}
}

func testBulkMessage(s *testSignaler, data []outputs.Data) message {
	return message{context: Context{Signal: s}, data: data}
}
