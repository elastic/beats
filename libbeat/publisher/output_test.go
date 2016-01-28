package publisher

import (
	"testing"

	"github.com/urso/ucfg"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/stretchr/testify/assert"
)

// Outputer that writes events to a channel.
type testOutputer struct {
	events chan common.MapStr
}

var _ outputs.Outputer = &testOutputer{}

func (t *testOutputer) Close() error {
	return nil
}

// PublishEvent writes events to a channel then calls Completed on trans.
// It always returns nil.
func (t *testOutputer) PublishEvent(trans outputs.Signaler, opts outputs.Options,
	event common.MapStr) error {
	t.events <- event
	outputs.SignalCompleted(trans)
	return nil
}

// Test OutputWorker by calling onStop() and onMessage() with various inputs.
func TestOutputWorker(t *testing.T) {
	outputer := &testOutputer{events: make(chan common.MapStr, 10)}
	ow := newOutputWorker(
		ucfg.New(),
		outputer,
		common.NewWorkerSignal(),
		1, 0)

	ow.onStop() // Noop

	var testCases = []message{
		testMessage(newTestSignaler(), nil),
		testMessage(newTestSignaler(), testEvent()),
		testBulkMessage(newTestSignaler(), []common.MapStr{testEvent()}),
	}

	for _, m := range testCases {
		sig := m.context.Signal.(*testSignaler)
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
