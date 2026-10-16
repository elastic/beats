// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"testing"
	"time"

	eventhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// TestProcessEventsConcurrentConversion verifies that the events published
// from one Event Hub message can be read by the output while the input is
// still publishing the remaining records of the same message.
//
// See https://github.com/elastic/beats/issues/53816
func TestProcessEventsConcurrentConversion(t *testing.T) {
	log := logptest.NewTestingLogger(t, inputName)
	metrics := newInputMetrics(monitoring.NewRegistry(), log)
	client := newConversionClient()
	input := eventHubInputV1{
		config:         defaultTestConfig,
		log:            log,
		metrics:        metrics,
		pipelineClient: client,
		messageDecoder: messageDecoder{config: defaultTestConfig, log: log, metrics: metrics},
	}

	var sequence, offset int64 = 12, 1234
	now := time.Now()
	message := &eventhub.Event{
		Data: recordsPayload(10000),
		SystemProperties: &eventhub.SystemProperties{
			SequenceNumber: &sequence,
			Offset:         &offset,
			EnqueuedTime:   &now,
		},
	}

	input.processEvents(message)

	assert.NoError(t, client.wait(), "converting a published event should not fail")
	assert.Equal(t, 10000, client.published, "the message should expand into one event per record")
}

// TestProcessEventsIndependentMetadata verifies that each event gets its own
// `azure` map, so outputs can read it while the input publishes more events.
func TestProcessEventsIndependentMetadata(t *testing.T) {
	log := logptest.NewTestingLogger(t, inputName)
	metrics := newInputMetrics(monitoring.NewRegistry(), log)
	client := &fakeClient{}
	input := eventHubInputV1{
		config:         defaultTestConfig,
		log:            log,
		metrics:        metrics,
		pipelineClient: client,
		messageDecoder: messageDecoder{config: defaultTestConfig, log: log, metrics: metrics},
	}

	var sequence, offset int64 = 12, 1234
	now := time.Now()
	input.processEvents(&eventhub.Event{
		Data: recordsPayload(2),
		SystemProperties: &eventhub.SystemProperties{
			SequenceNumber: &sequence,
			Offset:         &offset,
			EnqueuedTime:   &now,
		},
	})

	assertIndependentMetadata(t, client.publishedEvents)
}

func TestStripConnectionString(t *testing.T) {
	tests := []struct {
		connectionString, expected string
	}{
		{
			"Endpoint=sb://something",
			"(redacted)",
		},
		{
			"Endpoint=sb://dummynamespace.servicebus.windows.net/;SharedAccessKeyName=DummyAccessKeyName;SharedAccessKey=5dOntTRytoC24opYThisAsit3is2B+OGY1US/fuL3ly=",
			"Endpoint=sb://dummynamespace.servicebus.windows.net/",
		},
		{
			"Endpoint=sb://dummynamespace.servicebus.windows.net/;SharedAccessKey=5dOntTRytoC24opYThisAsit3is2B+OGY1US/fuL3ly=",
			"Endpoint=sb://dummynamespace.servicebus.windows.net/",
		},
	}

	for _, tt := range tests {
		res := stripConnectionString(tt.connectionString)
		assert.Equal(t, res, tt.expected)
	}
}
