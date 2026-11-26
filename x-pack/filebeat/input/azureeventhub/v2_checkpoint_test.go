// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	_ "github.com/elastic/beats/v7/libbeat/processors/actions"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/memqueue"
	"github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/pkg/mapstr"
	"github.com/elastic/beats/v7/pkg/monitoring"
)

// These tests use the real publisher pipeline, drop_event processor, input ACK
// callback, and ordered acker. Only the output and the Azure checkpoint writes
// are replaced.

// TestProcessReceivedEventsCoalescesFilteredCheckpoints verifies that a
// message whose records are all dropped by drop_event writes exactly one
// checkpoint, instead of one per dropped record.
func TestProcessReceivedEventsCoalescesFilteredCheckpoints(t *testing.T) {
	input := newTestInputV2(t)
	checkpoints := &checkpointRecorder{}
	client, output := newCheckpointTestClient(t, checkpoints, mapstr.M{})

	received := []*azeventhubs.ReceivedEventData{
		{EventData: azeventhubs.EventData{Body: recordsPayload(28)}, Offset: "100", SequenceNumber: 1},
		{EventData: azeventhubs.EventData{Body: recordsPayload(45)}, Offset: "200", SequenceNumber: 2},
	}
	require.NoError(t, input.processReceivedEvents(received, "0", client), "the receive batch should be processed")
	assert.Equal(t, []checkpointMetadata{
		{offset: "100", sequenceNumber: 1},
		{offset: "200", sequenceNumber: 2},
	}, checkpoints.snapshot(), "a fully filtered message should write exactly one checkpoint")
	assert.Empty(t, output.batches, "drop_event should prevent every record from reaching the output")
	assert.Equal(t, uint64(73), input.metrics.sentEvents.Get(), "all decoded records should pass through the pipeline")

	require.NoError(t, input.processReceivedEvents([]*azeventhubs.ReceivedEventData{
		{EventData: azeventhubs.EventData{Body: recordsPayload(3)}, Offset: "300", SequenceNumber: 3},
	}, "0", client), "the next receive batch should be processed")
	assert.Equal(t, []checkpointMetadata{
		{offset: "100", sequenceNumber: 1},
		{offset: "200", sequenceNumber: 2},
		{offset: "300", sequenceNumber: 3},
	}, checkpoints.snapshot(), "filtered-only traffic must keep advancing the checkpoint")
}

// TestProcessReceivedEventsWaitsForOutputACK verifies that a message is
// checkpointed only after all of its records are acknowledged by the output
// or filtered, including when the output acknowledges records of the same
// message in separate batches.
func TestProcessReceivedEventsWaitsForOutputACK(t *testing.T) {
	for _, tc := range []struct {
		name       string
		bodies     []string
		keptEvents int
		// completedAfterACK[i] is the sequence number expected to be
		// checkpointed after the i-th output ACK, or 0 for no new write.
		completedAfterACK []int64
	}{
		{
			name: "filtered prefix and published final record",
			bodies: []string{
				`{"records":[{"action":"drop"},{"action":"keep"}]}`,
			},
			keptEvents: 1, completedAfterACK: []int64{1},
		},
		{
			name: "published prefix and filtered final message",
			bodies: []string{
				`{"records":[{"action":"keep"}]}`,
				`{"records":[{"action":"drop"},{"action":"drop"}]}`,
			},
			// The filtered message completes behind the in-flight record.
			// One ACK releases both messages; only the newest is written.
			keptEvents: 1, completedAfterACK: []int64{2},
		},
		{
			name: "published records acknowledged separately",
			bodies: []string{
				`{"records":[{"action":"keep"},{"action":"keep"}]}`,
				`{"records":[{"action":"keep"},{"action":"drop"}]}`,
			},
			keptEvents: 3, completedAfterACK: []int64{0, 1, 2},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := newTestInputV2(t)
			checkpoints := &checkpointRecorder{}
			client, output := newCheckpointTestClient(t, checkpoints, mapstr.M{
				"when": mapstr.M{"contains": mapstr.M{"message": `"drop"`}},
			})
			var received []*azeventhubs.ReceivedEventData
			now := time.Now()
			for i, body := range tc.bodies {
				received = append(received, &azeventhubs.ReceivedEventData{
					EventData:      azeventhubs.EventData{Body: []byte(body)},
					Offset:         strconv.Itoa((i + 1) * 100),
					SequenceNumber: int64(i + 1),
					EnqueuedTime:   &now,
				})
			}
			require.NoError(t, input.processReceivedEvents(received, "0", client), "the receive batch should be processed")
			assert.Empty(t, checkpoints.snapshot(), "filtered records must not checkpoint past records waiting for the output")

			var expected []checkpointMetadata
			for i := range tc.keptEvents {
				select {
				case batch := <-output.batches:
					require.Len(t, batch.Events(), 1, "the output should acknowledge one record at a time")
					batch.ACK()
				case <-time.After(5 * time.Second):
					t.Fatal("a retained record did not reach the output")
				}
				select {
				case <-output.acks:
				case <-time.After(5 * time.Second):
					t.Fatal("an output acknowledgement did not reach the input")
				}
				if completed := tc.completedAfterACK[i]; completed > 0 {
					expected = append(expected, checkpointMetadata{offset: strconv.Itoa(int(completed) * 100), sequenceNumber: completed})
				}
				assert.Equal(t, expected, checkpoints.snapshot(), "only fully completed messages may be checkpointed")
			}
		})
	}
}

// TestProcessReceivedEventsIgnoresEmptyMessages verifies that messages that
// decode to no records neither publish events nor write checkpoints.
func TestProcessReceivedEventsIgnoresEmptyMessages(t *testing.T) {
	input := newTestInputV2(t)
	checkpoints := &checkpointRecorder{}
	client, _ := newCheckpointTestClient(t, checkpoints, mapstr.M{})
	received := []*azeventhubs.ReceivedEventData{
		{EventData: azeventhubs.EventData{Body: recordsPayload(2)}, Offset: "100", SequenceNumber: 1},
		{EventData: azeventhubs.EventData{Body: []byte(`{"records":[]}`)}, Offset: "200", SequenceNumber: 2},
	}
	require.NoError(t, input.processReceivedEvents(received, "0", client), "a trailing empty message should be processed")
	assert.Equal(t, []checkpointMetadata{{offset: "100", sequenceNumber: 1}}, checkpoints.snapshot(),
		"an empty message publishes nothing, so it cannot be checkpointed on its own")
	assert.Equal(t, uint64(2), input.metrics.processedMessages.Get(), "empty messages should still count as processed")
	assert.Equal(t, uint64(2), input.metrics.sentEvents.Get(), "only decoded records should be sent to the pipeline")

	require.NoError(t, input.processReceivedEvents(nil, "0", client), "an empty receive batch should be harmless")
	assert.Len(t, checkpoints.snapshot(), 1, "an empty receive batch must not add a checkpoint")
}

// TestProcessReceivedEventsConcurrentACKs verifies that output ACK callbacks
// and synchronous drop_event acknowledgements, which can overlap, never
// duplicate a checkpoint or move it backwards.
func TestProcessReceivedEventsConcurrentACKs(t *testing.T) {
	input := newTestInputV2(t)
	checkpoints := &checkpointRecorder{}
	client, output := newCheckpointTestClient(t, checkpoints, mapstr.M{
		"when": mapstr.M{"contains": mapstr.M{"message": `"drop"`}},
	})
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case batch := <-output.batches:
				batch.ACK()
			case <-output.acks:
			case <-stop:
				return
			}
		}
	}()
	defer func() { close(stop); <-done }()

	const messageCount = 40
	now := time.Now()
	for i := 1; i <= messageCount; i++ {
		require.NoError(t, input.processReceivedEvents([]*azeventhubs.ReceivedEventData{
			{
				EventData: azeventhubs.EventData{Body: []byte(`{"records":[{"action":"keep"},{"action":"drop"},{"action":"keep"},{"action":"drop"}]}`)},
				Offset:    strconv.Itoa(i * 100), SequenceNumber: int64(i), EnqueuedTime: &now,
			},
		}, "0", client), "publishing a message concurrently with output ACKs should succeed")
	}
	require.Eventually(t, func() bool {
		got := checkpoints.snapshot()
		return len(got) > 0 && got[len(got)-1].sequenceNumber == messageCount
	}, 5*time.Second, time.Millisecond, "the last message should eventually be checkpointed")
	got := checkpoints.snapshot()
	for i := 1; i < len(got); i++ {
		assert.Greater(t, got[i].sequenceNumber, got[i-1].sequenceNumber,
			"overlapping callbacks must not duplicate checkpoints or move them backwards")
	}
}

func newCheckpointTestClient(t *testing.T, checkpoints *checkpointRecorder, dropConfig mapstr.M) (beat.Client, *checkpointTestOutput) {
	t.Helper()
	// Queue-reader teardown can log after Disconnect returns. Do not bind
	// these background log writes to the lifetime of the test's logger.
	logger := logp.NewNopLogger()
	info := beat.Info{Logger: logger}
	support, err := processing.MakeDefaultSupport(false, nil)(info, logger, config.MustNewConfigFrom(mapstr.M{
		"processors": []any{mapstr.M{"drop_event": dropConfig}},
	}))
	require.NoError(t, err, "the drop_event processor should initialize")
	t.Cleanup(func() { assert.NoError(t, support.Close(), "the processor support should close") })

	output := &checkpointTestOutput{batches: make(chan publisher.Batch, 128), acks: make(chan int, 128)}
	p, err := pipeline.New(info, pipeline.Monitors{Logger: logger, Metrics: monitoring.NewRegistry()}, config.Namespace{}, outputs.Group{
		Clients: []outputs.Client{output}, BatchSize: 1,
		QueueFactory: memqueue.FactoryForSettings[publisher.Event](memqueue.Settings{
			Events: 128, MaxGetRequest: 1,
		}),
	}, pipeline.Settings{Processors: support, WaitClose: time.Second, WaitCloseMode: pipeline.WaitOnPipelineCloseThenForce})
	require.NoError(t, err, "the publisher pipeline should initialize")
	t.Cleanup(func() { assert.NoError(t, p.Disconnect(context.Background()), "the pipeline should disconnect") })
	client, err := initializePartitionResources(t.Context(), checkpoints, checkpointTestPipeline{Pipeline: p, acks: output.acks}, logger)
	require.NoError(t, err, "the partition pipeline client should connect")
	t.Cleanup(func() { assert.NoError(t, client.Close(), "the pipeline client should close") })
	return client, output
}

// checkpointRecorder replaces the Azure partition client and records every
// checkpoint write.
type checkpointRecorder struct {
	mu          sync.Mutex
	checkpoints []checkpointMetadata
}

func (*checkpointRecorder) PartitionID() string { return "0" }

func (r *checkpointRecorder) UpdateCheckpoint(_ context.Context, event *azeventhubs.ReceivedEventData, _ *azeventhubs.UpdateCheckpointOptions) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checkpoints = append(r.checkpoints, checkpointMetadata{offset: event.Offset, sequenceNumber: event.SequenceNumber})
	return nil
}

func (r *checkpointRecorder) snapshot() []checkpointMetadata {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]checkpointMetadata(nil), r.checkpoints...)
}

// checkpointTestOutput hands each batch to the test, which acknowledges it
// explicitly to control when the input's ACK callback runs.
type checkpointTestOutput struct {
	batches chan publisher.Batch
	acks    chan int
}

func (*checkpointTestOutput) String() string { return "checkpoint_test" }
func (*checkpointTestOutput) Close() error   { return nil }
func (o *checkpointTestOutput) Publish(_ context.Context, batch publisher.Batch) error {
	o.batches <- batch
	return nil
}

// checkpointTestPipeline wraps the input's event listener so the test can
// observe when the pipeline has finished running an ACK callback. The input's
// checkpoint write runs synchronously inside ACKEvents, so a receive on acks
// means the corresponding checkpoint write, if any, has already happened.
type checkpointTestPipeline struct {
	beat.Pipeline
	acks chan int
}

func (p checkpointTestPipeline) ConnectWith(cfg beat.ClientConfig) (beat.Client, error) {
	cfg.EventListener = checkpointACKObserver{EventListener: cfg.EventListener, acks: p.acks}
	return p.Pipeline.ConnectWith(cfg)
}

type checkpointACKObserver struct {
	beat.EventListener
	acks chan int
}

func (o checkpointACKObserver) ACKEvents(n int) {
	o.EventListener.ACKEvents(n)
	o.acks <- n
}
