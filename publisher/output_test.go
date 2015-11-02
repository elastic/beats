package publisher

import (
	"testing"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/outputs"
	"github.com/stretchr/testify/assert"
)

// Outputer that writes events to a channel.
type testOutputer struct {
	events chan common.MapStr
}

var _ outputs.Outputer = &testOutputer{}

// PublishEvent writes events to a channel then calls Completed on trans.
// It always returns nil.
func (t *testOutputer) PublishEvent(trans outputs.Signaler, ts time.Time,
	event common.MapStr) error {
	t.events <- event
	outputs.SignalCompleted(trans)
	return nil
}

// Test OutputWorker by calling onStop() and onMessage() with various inputs.
func TestOutputWorker(t *testing.T) {
	outputer := &testOutputer{events: make(chan common.MapStr, 10)}
	ow := newOutputWorker(
		outputs.MothershipConfig{},
		outputer,
		newWorkerSignal(),
		1)

	ow.onStop() // Noop

	var testCases = []message{
		testMessage(newTestSignaler(), nil),
		testMessage(newTestSignaler(), testEvent()),
		testBulkMessage(newTestSignaler(), []common.MapStr{testEvent()}),
	}

	for _, m := range testCases {
		sig := m.context.signal.(*testSignaler)
		ow.onMessage(m)
		assert.True(t, sig.wait())

		if m.event != nil {
			assert.Equal(t, m.event, <-outputer.events)
		} else {
			for _, e := range m.events {
				assert.Equal(t, e, <-outputer.events)
			}
		}
	}
}
