// +build !integration

package publisher

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/stretchr/testify/assert"
)

// Outputer that writes events to a channel.
type testOutputer struct {
	data chan outputs.Data
}

var _ outputs.Outputer = &testOutputer{}

func (t *testOutputer) Close() error {
	return nil
}

// PublishEvent writes events to a channel then calls Completed on trans.
// It always returns nil.
func (t *testOutputer) PublishEvent(
	trans op.Signaler,
	_ outputs.Options,
	data outputs.Data,
) error {
	t.data <- data
	op.SigCompleted(trans)
	return nil
}

// Test OutputWorker by calling onStop() and onMessage() with various inputs.
func TestOutputWorker(t *testing.T) {
	outputer := &testOutputer{data: make(chan outputs.Data, 10)}
	ow := newOutputWorker(
		common.NewConfig(),
		outputer,
		newWorkerSignal(),
		1, 0)

	ow.onStop() // Noop

	var testCases = []message{
		testMessage(newTestSignaler(), outputs.Data{}),
		testMessage(newTestSignaler(), testEvent()),
		testBulkMessage(newTestSignaler(), []outputs.Data{testEvent()}),
	}

	for _, m := range testCases {
		sig := m.context.Signal.(*testSignaler)
		ow.onMessage(m)
		assert.True(t, sig.wait())

		if m.datum.Event != nil {
			assert.Equal(t, m.datum, <-outputer.data)
		} else {
			for _, e := range m.data {
				assert.Equal(t, e, <-outputer.data)
			}
		}
	}
}
