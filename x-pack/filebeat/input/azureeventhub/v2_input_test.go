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

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	inputv2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestRunUpdatesStatusToStartingAndFailed(t *testing.T) {
	input, err := newEventHubInputV2(azureInputConfig{}, logp.L())
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
		Logger:          logp.L(),
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

	log := logp.L()
	metrics := newInputMetrics(monitoring.NewRegistry(), log)

	sanitizers, err := newSanitizers(inputConfig.Sanitizers, inputConfig.LegacySanitizeOptions)
	require.NoError(t, err)

	input := &eventHubInputV2{
		config:  inputConfig,
		log:     log,
		metrics: metrics,
		messageDecoder: messageDecoder{
			config:     inputConfig,
			metrics:    metrics,
			log:        log,
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
			Offset:       0,
		},
		{
			EventData:    azeventhubs.EventData{Body: []byte(`{"records":[{"msg":"two"}]}`)},
			EnqueuedTime: &now,
			PartitionKey: &partitionKey,
			Offset:       1,
		},
		{
			EventData:    azeventhubs.EventData{Body: []byte(`{"records":[{"msg":"three"}]}`)},
			EnqueuedTime: &now,
			PartitionKey: &partitionKey,
			Offset:       2,
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
