package stress

import (
	"math/rand"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/publisher"
)

type testOutput struct {
	config     testOutputConfig
	observer   outputs.Observer
	batchCount int
}

type testOutputConfig struct {
	Worker      int           `config:"worker" validate:"min=1"`
	BulkMaxSize int           `config:"bulk_max_size"`
	Retry       int           `config:"retry"`
	MinWait     time.Duration `config:"min_wait"`
	MaxWait     time.Duration `config:"max_wait"`
	Fail        struct {
		EveryBatch int
	}
}

var defaultTestOutputConfig = testOutputConfig{
	Worker:      1,
	BulkMaxSize: 64,
}

func init() {
	outputs.RegisterType("test", makeTestOutput)
}

func makeTestOutput(beat beat.Info, observer outputs.Observer, cfg *common.Config) (outputs.Group, error) {
	config := defaultTestOutputConfig
	if err := cfg.Unpack(&config); err != nil {
		return outputs.Fail(err)
	}

	clients := make([]outputs.Client, config.Worker)
	for i := range clients {
		client := &testOutput{config: config, observer: observer}
		clients[i] = client
	}

	return outputs.Success(config.BulkMaxSize, config.Retry, clients...)
}

func (*testOutput) Close() error { return nil }

func (t *testOutput) Publish(batch publisher.Batch) error {
	config := &t.config

	n := len(batch.Events())
	t.observer.NewBatch(n)

	min := int64(config.MinWait)
	max := int64(config.MaxWait)
	if max > 0 && min < max {
		waitFor := rand.Int63n(max-min) + min

		// TODO: make wait interruptable via `Close`
		time.Sleep(time.Duration(waitFor))
	}

	// fail complete batch
	if config.Fail.EveryBatch > 0 {
		t.batchCount++

		if config.Fail.EveryBatch == t.batchCount {
			t.batchCount = 0
			t.observer.Failed(n)
			batch.Retry()
			return nil
		}

	}

	// TODO: add support to fail single events at end of batch or randomly

	// ack complete batch
	batch.ACK()
	t.observer.Acked(n)

	return nil
}
