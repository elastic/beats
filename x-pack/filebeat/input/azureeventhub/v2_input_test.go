// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	inputv2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestRunUpdatesStatusToStartingAndFailed(t *testing.T) {
	logger := logptest.NewTestingLogger(t, inputName)
	input, err := newEventHubInputV2(azureInputConfig{}, logger)
	require.NoError(t, err)

	eventHubInputV2, ok := input.(*eventHubInputV2)
	require.True(t, ok)

	// Mock the setup function to return an error, so the Run function exits early.
	// This allows us to test the initial status update without running the full input.
	eventHubInputV2.setupFn = func(ctx context.Context) error {
		return errors.New("setup failure")
	}

	ctx := t.Context()

	statusReporter := newMockStatusReporter()
	inputTestCtx := inputv2.Context{
		Logger:          logger,
		Cancelation:     ctx,
		MetricsRegistry: monitoring.NewRegistry(),
	}
	inputTestCtx = inputTestCtx.WithStatusReporter(statusReporter)

	// The Run function is expected to return the error from the mock setup function.
	err = eventHubInputV2.Run(inputTestCtx, nil)
	require.Error(t, err, "setup failure")

	// Verify that the status was updated to Starting and then to Failed
	assert.Len(t, statusReporter.statuses, 3)
	assert.Equal(t, status.Starting, statusReporter.statuses[0])
	assert.Equal(t, status.Configuring, statusReporter.statuses[1])
	assert.Equal(t, status.Failed, statusReporter.statuses[2])
}

func TestProcessReceivedEventsUpdatesProcessingTimeOnce(t *testing.T) {
	// This test verifies that processingTime is updated exactly once
	// per call to processReceivedEvents, regardless of the number of
	// events processed. Before the fix, processingTime was updated
	// inside the loop, resulting in N updates for N events.

	inputConfig := azureInputConfig{
		EventHubName:  "test-eventhub",
		ConsumerGroup: "test-consumer-group",
	}

	logger := logptest.NewTestingLogger(t, inputName)
	metrics := newInputMetrics(monitoring.NewRegistry(), logger)

	sanitizers, err := newSanitizers(inputConfig.Sanitizers, inputConfig.LegacySanitizeOptions)
	require.NoError(t, err)

	input := &eventHubInputV2{
		config:  inputConfig,
		log:     logger,
		metrics: metrics,
		messageDecoder: messageDecoder{
			config:     inputConfig,
			metrics:    metrics,
			log:        logger,
			sanitizers: sanitizers,
		},
	}

	now := time.Now()
	partitionKey := "test-key"

	// Create multiple received events so we can verify
	// processingTime is updated once, not per event.
	receivedEvents := []*azeventhubs.ReceivedEventData{
		{
			EventData:    azeventhubs.EventData{Body: []byte(`{"records":[{"msg":"one"}]}`)},
			EnqueuedTime: &now,
			PartitionKey: &partitionKey,
			Offset:       "0",
		},
		{
			EventData:    azeventhubs.EventData{Body: []byte(`{"records":[{"msg":"two"}]}`)},
			EnqueuedTime: &now,
			PartitionKey: &partitionKey,
			Offset:       "1",
		},
		{
			EventData:    azeventhubs.EventData{Body: []byte(`{"records":[{"msg":"three"}]}`)},
			EnqueuedTime: &now,
			PartitionKey: &partitionKey,
			Offset:       "2",
		},
	}

	fakeClient := &fakeClient{}

	err = input.processReceivedEvents(receivedEvents, "0", fakeClient)
	require.NoError(t, err)

	// Verify all 3 messages were processed.
	assert.Equal(t, uint64(3), metrics.processedMessages.Get())

	// Verify processingTime was updated exactly once (after the loop),
	// not once per event (which was the bug).
	assert.Equal(t, 1, metrics.processingTime.Size(),
		"processingTime should be updated once per call, not once per event")
}

func TestProcessReceivedEventsRetainsOnlyCheckpointMetadata(t *testing.T) {
	now := time.Now()
	logger := logptest.NewTestingLogger(t, inputName)
	metrics := newInputMetrics(monitoring.NewRegistry(), logger)
	sanitizers, err := newSanitizers(defaultTestConfig.Sanitizers, defaultTestConfig.LegacySanitizeOptions)
	require.NoError(t, err, "sanitizers should initialize")
	input := &eventHubInputV2{
		config:  defaultTestConfig,
		log:     logger,
		metrics: metrics,
		messageDecoder: messageDecoder{
			config: defaultTestConfig, log: logger, metrics: metrics, sanitizers: sanitizers,
		},
	}
	received := []*azeventhubs.ReceivedEventData{
		{
			EventData: azeventhubs.EventData{
				Body:       []byte(`{"records":[{"msg":"one"},{"msg":"two"}]}`),
				Properties: map[string]any{"application": "test"},
			},
			Offset:           "1234",
			SequenceNumber:   42,
			EnqueuedTime:     &now,
			RawAMQPMessage:   &azeventhubs.AMQPAnnotatedMessage{},
			SystemProperties: map[string]any{"service": "test"},
		},
		{
			EventData:      azeventhubs.EventData{Body: []byte(`{"msg":"three"}`)},
			Offset:         "5678",
			SequenceNumber: 43,
		},
	}
	client := &fakeClient{}
	require.NoError(t, input.processReceivedEvents(received, "0", client), "messages should be published")
	require.Len(t, client.publishedEvents, 3, "the first message should expand into two records")

	for i, source := range []*azeventhubs.ReceivedEventData{received[0], received[0], received[1]} {
		checkpoint, ok := client.publishedEvents[i].Private.(*checkpointMetadata)
		require.True(t, ok, "event %d should carry checkpoint metadata accepted by the ACK callback", i)
		assert.Equal(t, &checkpointMetadata{
			offset: source.Offset, sequenceNumber: source.SequenceNumber,
		}, checkpoint, "event %d should retain only checkpoint metadata", i)
	}
	assert.Same(t, client.publishedEvents[0].Private, client.publishedEvents[1].Private,
		"records from the same message should share checkpoint metadata")
}

// TestProcessReceivedEventsConcurrentConversion verifies that the events
// published from one Event Hub message can be read by the output while the
// input is still publishing the remaining records of the same message.
//
// See https://github.com/elastic/beats/issues/53816
func TestProcessReceivedEventsConcurrentConversion(t *testing.T) {
	input := newTestInputV2(t)
	client := newConversionClient()

	err := input.processReceivedEvents(receivedEventsWithRecords(10000), "0", client)
	require.NoError(t, err, "processing the received events should not fail")

	assert.NoError(t, client.wait(), "converting a published event should not fail")
	assert.Equal(t, 10000, client.published, "the message should expand into one event per record")
}

// TestProcessReceivedEventsIndependentMetadata verifies that each event gets
// its own `azure` map, so processors and outputs can change or read it
// without affecting the other events from the same message.
func TestProcessReceivedEventsIndependentMetadata(t *testing.T) {
	input := newTestInputV2(t)
	client := &fakeClient{}

	receivedEvents := receivedEventsWithRecords(2)
	err := input.processReceivedEvents(receivedEvents, "0", client)
	require.NoError(t, err, "processing the received events should not fail")

	received := receivedEvents[0]
	assertIndependentMetadata(t, client.publishedEvents, mapstr.M{
		"partition_id":    "0",
		"eventhub":        defaultTestConfig.EventHubName,
		"consumer_group":  defaultTestConfig.ConsumerGroup,
		"offset":          received.Offset,
		"sequence_number": received.SequenceNumber,
		"enqueued_time":   received.EnqueuedTime,
		"partition_key":   *received.PartitionKey,
	})
}

// newTestInputV2 returns a v2 input ready to process received events.
func newTestInputV2(t *testing.T) *eventHubInputV2 {
	t.Helper()
	logger := logptest.NewTestingLogger(t, inputName)
	metrics := newInputMetrics(monitoring.NewRegistry(), logger)
	sanitizers, err := newSanitizers(defaultTestConfig.Sanitizers, defaultTestConfig.LegacySanitizeOptions)
	require.NoError(t, err, "creating the default sanitizers should not fail")

	return &eventHubInputV2{
		config:  defaultTestConfig,
		log:     logger,
		metrics: metrics,
		messageDecoder: messageDecoder{
			config:     defaultTestConfig,
			metrics:    metrics,
			log:        logger,
			sanitizers: sanitizers,
		},
	}
}

// receivedEventsWithRecords returns a single received event whose body
// contains n records.
func receivedEventsWithRecords(n int) []*azeventhubs.ReceivedEventData {
	now := time.Now()
	partitionKey := "key"
	return []*azeventhubs.ReceivedEventData{
		{
			EventData:      azeventhubs.EventData{Body: recordsPayload(n)},
			SequenceNumber: 12,
			EnqueuedTime:   &now,
			PartitionKey:   &partitionKey,
			Offset:         "1234",
		},
	}
}

// mockStatusReporter is a mock implementation of the status.Reporter interface.
// It is used to verify that the input updates its status correctly.
type mockStatusReporter struct {
	mutex    sync.Mutex
	statuses []status.Status
}

func newMockStatusReporter() *mockStatusReporter {
	return &mockStatusReporter{}
}

func (r *mockStatusReporter) UpdateStatus(status status.Status, msg string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.statuses = append(r.statuses, status)
}
