// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package api

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/logp"
)

type memoryAuthClient struct {
	Requests chan []EventRequest
	Err      error
}

func (m *memoryAuthClient) SendEvents(requests []EventRequest) error {
	if m.Err != nil {
		return m.Err
	}

	m.Requests <- requests
	return nil
}

func (m *memoryAuthClient) Close() {
	close(m.Requests)
}

func (m *memoryAuthClient) Configuration() (ConfigBlocks, error) {
	return ConfigBlocks{}, nil
}

func newMemoryAuthClient() *memoryAuthClient {
	return &memoryAuthClient{Requests: make(chan []EventRequest)}
}

func TestReporterReportEvents(t *testing.T) {
	t.Run("single request", testBatch(1, 100))
	t.Run("receive all events when the requests size is exactly the batch size", testBatch(100, 100))
	t.Run("receive all events when events are send in multiple batch", testBatch(1234, 25))
}

func testBatch(numberOfEvents, maxBatchSize int) func(*testing.T) {
	return func(t *testing.T) {
		event := &testEvent{Message: "OK"}
		client := newMemoryAuthClient()
		defer client.Close()
		reporter := NewEventReporter(logp.NewLogger(""), client, 1*time.Second, maxBatchSize)
		reporter.Start()
		defer reporter.Stop()

		go func() {
			for i := 0; i < numberOfEvents; i++ {
				reporter.AddEvent(event)
			}
		}()

		var receivedEvents int
		expectedbatch := int(math.Ceil(float64(numberOfEvents) / float64(maxBatchSize)))

		for receivedBatchs := 0; receivedBatchs < expectedbatch; receivedBatchs++ {
			requests := <-client.Requests
			receivedEvents += len(requests)
		}

		assert.Equal(t, numberOfEvents, receivedEvents)
	}
}
