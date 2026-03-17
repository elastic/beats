package monitors

import (
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDeferredPipeline(t *testing.T) {
	done := make(chan struct{})
	mock := &MockPipeline{}
	pipeline := WithDeferredPipelineClose(mock, done)
	pipeline.Connect()

	// Closing the pipeline shouldn't close the wrapped pipeline
	pipeline.(io.Closer).Close()
	assert.False(t, mock.Clients[0].closed)

	// But closing the channel should close the wrapped pipeline
	close(done)
	assert.Eventually(
		t,
		func() bool { return mock.Clients[0].closed },
		time.Second,
		time.Millisecond*50,
	)
}
