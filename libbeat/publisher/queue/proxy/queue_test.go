package proxyqueue

import (
	"testing"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestQueueStuff(t *testing.T) {
	logger := logp.NewLogger("proxy-queue-tests")
	testQueue := NewQueue(logger, Settings{BatchSize: 2})
}
