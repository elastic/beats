// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package shipper

import (
	"context"
	"errors"
	"fmt"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
	"github.com/elastic/beats/v7/libbeat/outputs/shipper/api"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-shipper-client/pkg/helpers"
	pb "github.com/elastic/elastic-agent-shipper-client/pkg/proto"
	"github.com/elastic/elastic-agent-shipper-client/pkg/proto/messages"
)

func TestToShipperEvent(t *testing.T) {
	ts := time.Now().Truncate(time.Second)

	cases := []struct {
		name   string
		value  publisher.Event
		exp    *messages.Event
		expErr string
	}{
		{
			name: "successfully converts an event without source and data stream",
			value: publisher.Event{
				Content: beat.Event{
					Timestamp: ts,
					Meta: mapstr.M{
						"metafield": 42,
					},
					Fields: mapstr.M{
						"field": "117",
					},
				},
			},
			exp: &messages.Event{
				Timestamp:  timestamppb.New(ts),
				Source:     &messages.Source{},
				DataStream: &messages.DataStream{},
				Metadata: protoStruct(t, map[string]interface{}{
					"metafield": 42,
				}),
				Fields: protoStruct(t, map[string]interface{}{
					"field": "117",
				}),
			},
		},
		{
			name: "successfully converts an event with source and data stream",
			value: publisher.Event{
				Content: beat.Event{
					Timestamp: ts,
					Meta: mapstr.M{
						"metafield": 42,
						"input_id":  "input",
						"stream_id": "stream",
					},
					Fields: mapstr.M{
						"field": "117",
						"data_stream": mapstr.M{
							"type":      "ds-type",
							"namespace": "ds-namespace",
							"dataset":   "ds-dataset",
						},
					},
				},
			},
			exp: &messages.Event{
				Timestamp: timestamppb.New(ts),
				Source: &messages.Source{
					InputId:  "input",
					StreamId: "stream",
				},
				DataStream: &messages.DataStream{
					Type:      "ds-type",
					Namespace: "ds-namespace",
					Dataset:   "ds-dataset",
				},
				Metadata: protoStruct(t, map[string]interface{}{
					"metafield": 42,
					"input_id":  "input",
					"stream_id": "stream",
				}),
				Fields: protoStruct(t, map[string]interface{}{
					"field": "117",
					"data_stream": map[string]interface{}{
						"type":      "ds-type",
						"namespace": "ds-namespace",
						"dataset":   "ds-dataset",
					},
				}),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			converted, err := toShipperEvent(tc.value)
			if tc.expErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErr)
				require.Nil(t, converted)
				return
			}
			requireEqualProto(t, tc.exp, converted)
		})
	}
}

func TestPublish(t *testing.T) {
	events := []beat.Event{
		{
			Meta:   mapstr.M{"event": "first"},
			Fields: mapstr.M{"a": "b"},
		},
		{
			Meta:   nil, // see failMarshal()
			Fields: mapstr.M{"a": "b"},
		},
		{
			Meta:   mapstr.M{"event": "third"},
			Fields: mapstr.M{"e": "f"},
		},
	}

	cases := []struct {
		name        string
		events      []beat.Event
		expSignals  []outest.BatchSignal
		serverError error
		expError    string
		// note: this sets the queue size used by the mock output
		// if the mock shipper receives more than this count of events, the test will fail
		qSize            int
		observerExpected *TestObserver
		marshalMethod    func(e publisher.Event) (*messages.Event, error)
	}{
		{
			name:          "sends a batch",
			events:        events,
			marshalMethod: toShipperEvent,
			expSignals: []outest.BatchSignal{
				{Tag: outest.BatchACK},
			},
			qSize:            3,
			observerExpected: &TestObserver{batch: 3, acked: 3},
		},
		{
			name:   "retries not accepted events",
			events: events,
			expSignals: []outest.BatchSignal{
				{Tag: outest.BatchACK},
			},
			marshalMethod:    failMarshal, // emulate a dropped event
			qSize:            3,
			observerExpected: &TestObserver{batch: 3, dropped: 1, acked: 2},
		},
		{
			name:   "cancels the batch if server error",
			events: events,
			expSignals: []outest.BatchSignal{
				{Tag: outest.BatchCancelled},
			},
			marshalMethod:    toShipperEvent,
			qSize:            3,
			observerExpected: &TestObserver{cancelled: 3, batch: 3},
			serverError:      errors.New("some error"),
			expError:         "failed to publish the batch to the shipper, none of the 3 events were accepted",
		},
		{
			name:   "splits the batch on resource exceeded error",
			events: events,
			expSignals: []outest.BatchSignal{
				{Tag: outest.BatchSplitRetry},
			},
			marshalMethod:    toShipperEvent,
			qSize:            3,
			observerExpected: &TestObserver{batch: 3, split: 1},
			serverError:      status.Error(codes.ResourceExhausted, "rpc size limit exceeded"),
		},
		{
			name:   "drops an unsplittable batch on resource exceeded error",
			events: events[:1], // only 1 event so SplitRetry returns false
			expSignals: []outest.BatchSignal{
				{Tag: outest.BatchSplitRetry},
				{Tag: outest.BatchDrop},
			},
			marshalMethod:    toShipperEvent,
			qSize:            1,
			observerExpected: &TestObserver{batch: 1, dropped: 1},
			serverError:      status.Error(codes.ResourceExhausted, "rpc size limit exceeded"),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.marshalMethod != nil {
				shipperProcessor = tc.marshalMethod
			}
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			addr, producer, stop := runServer(
				t, tc.qSize, constErrorCallback(tc.serverError), "localhost:0")
			defer stop()

			cfg, err := config.NewConfigFrom(map[string]interface{}{
				"server": addr,
			})
			require.NoError(t, err)
			observer := &TestObserver{}

			client := createShipperClient(t, cfg, observer)

			batch := outest.NewBatch(tc.events...)

			err = client.Publish(ctx, batch)
			if tc.expError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expError)
			} else {
				require.NoError(t, err)
				producer.Persist(uint64(tc.qSize)) // always persisted all published events
			}

			assert.Eventually(t, func() bool {
				// there is a background routine that checks acknowledgments,
				// it should eventually change the status of the batch
				return reflect.DeepEqual(tc.expSignals, batch.Signals)
			}, 100*time.Millisecond, 10*time.Millisecond)
			require.Equal(t, tc.expSignals, batch.Signals)
			if tc.observerExpected != nil {
				require.Equal(t, tc.observerExpected, observer)
			}
		})
	}
	// reset marshaler
	shipperProcessor = toShipperEvent

	t.Run("cancels the batch when a different server responds", func(t *testing.T) {
		t.Skip("Flaky test: https://github.com/elastic/beats/issues/34984")
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		addr, _, stop := runServer(t, 5, nil, "localhost:0")
		defer stop()

		cfg, err := config.NewConfigFrom(map[string]interface{}{
			"server":  addr,
			"timeout": 5, // 5 sec
			"backoff": map[string]interface{}{
				"init": "10ms",
				"max":  "5s",
			},
		})
		require.NoError(t, err)
		observer := &TestObserver{}
		client := createShipperClient(t, cfg, observer)

		// Should accept the batch and put it to the pending list
		batch := outest.NewBatch(events...)
		err = client.Publish(ctx, batch)
		require.NoError(t, err)

		// Replace the server (would change the ID)
		stop()

		_, _, stop = runServer(t, 5, nil, addr)
		defer stop()
		err = client.Connect()
		require.NoError(t, err)

		expSignals := []outest.BatchSignal{
			{
				Tag: outest.BatchCancelled,
			},
		}
		assert.Eventually(t, func() bool {
			// there is a background routine that checks acknowledgments,
			// it should eventually cancel the batch because the IDs don't match
			return reflect.DeepEqual(expSignals, batch.Signals)
		}, 100*time.Millisecond, 10*time.Millisecond)
		require.Equal(t, expSignals, batch.Signals)
	})

	t.Run("acks multiple batches", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		addr, producer, stop := runServer(t, 9, nil, "localhost:0")
		defer stop()

		cfg, err := config.NewConfigFrom(map[string]interface{}{
			"server":  addr,
			"timeout": 5, // 5 sec
			"backoff": map[string]interface{}{
				"init": "10ms",
				"max":  "5s",
			},
		})
		require.NoError(t, err)
		observer := &TestObserver{}
		expectedObserver := &TestObserver{batch: 9, acked: 9}
		client := createShipperClient(t, cfg, observer)

		// Should accept the batch and put it to the pending list
		batch1 := outest.NewBatch(events...)
		err = client.Publish(ctx, batch1)
		require.NoError(t, err)

		batch2 := outest.NewBatch(events...)
		err = client.Publish(ctx, batch2)
		require.NoError(t, err)

		batch3 := outest.NewBatch(events...)
		err = client.Publish(ctx, batch3)
		require.NoError(t, err)

		expSignals := []outest.BatchSignal{
			{
				Tag: outest.BatchACK,
			},
		}

		producer.Persist(9) // 2 events per batch, 3 batches

		assert.Eventually(t, func() bool {
			// there is a background routine that checks acknowledgments,
			// it should eventually send expected signals
			return reflect.DeepEqual(expSignals, batch1.Signals) &&
				reflect.DeepEqual(expSignals, batch2.Signals) &&
				reflect.DeepEqual(expSignals, batch3.Signals)
		}, 100*time.Millisecond, 10*time.Millisecond)
		require.Equal(t, expSignals, batch1.Signals, "batch1")
		require.Equal(t, expSignals, batch2.Signals, "batch2")
		require.Equal(t, expSignals, batch3.Signals, "batch3")
		require.Equal(t, expectedObserver, observer)
	})

	t.Run("live batches where all events are too large to ingest", func(t *testing.T) {
		// This tests recursive retry using live `ttlBatch` structs instead of mocks
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		errCallback := constErrorCallback(status.Error(codes.ResourceExhausted, "rpc size limit exceeded"))
		addr, _, stop := runServer(t, 9, errCallback, "localhost:0")
		defer stop()
		cfg, err := config.NewConfigFrom(map[string]interface{}{
			"server": addr,
		})
		require.NoError(t, err)
		observer := &TestObserver{}

		client := createShipperClient(t, cfg, observer)

		// Since we retry directly instead of going through a live pipeline,
		// the Publish call is synchronous and we can track state by modifying
		// local variables directly.
		retryCount := 0
		done := false
		batch := pipeline.NewBatchForTesting(
			[]publisher.Event{
				{Content: events[0]}, {Content: events[1]}, {Content: events[2]},
			},
			func(b publisher.Batch) {
				// Retry by sending directly back to Publish. In a live
				// pipeline, this would be sent through eventConsumer first
				// before calling Publish on the next free output worker.
				retryCount++
				err := client.Publish(ctx, b)
				assert.NoError(t, err, "Publish shouldn't return an error")
			},
			func() { done = true },
		)
		err = client.Publish(ctx, batch)
		assert.NoError(t, err, "Publish shouldn't return an error")

		// For three events there should be four retries in total:
		// {[event1], [event2, event3]}, then {[event2], [event3]}.
		// "done" should be true because after splitting into individual
		// events, all 3 will fail and be dropped.
		assert.Equal(t, 4, retryCount, "three-event batch should produce four total retries")
		assert.True(t, done, "batch should be done after Publish")

		// "batch" adds up all events passed into publish, including repeats,
		// so it should be 3 + 2 + 1 + 1 + 1 = 8
		expectedObserver := &TestObserver{split: 2, dropped: 3, batch: 8}
		require.Equal(t, expectedObserver, observer)
	})

	t.Run("live batches where only one event is too large to ingest", func(t *testing.T) {
		// This tests retry using live `ttlBatch` structs instead of mocks,
		// where one event is too large too ingest but the others are ok.
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		errCallback := func(batchEvents []*messages.Event) error {
			// Treat only the first event (which contains the metadata
			// string "first") as too large to ingest, and accept otherwise.
			for _, e := range batchEvents {
				if strings.Contains(e.String(), "\"first\"") {
					return status.Error(codes.ResourceExhausted, "rpc size limit exceeded")
				}
			}
			return nil
		}
		addr, _, stop := runServer(t, 9, errCallback, "localhost:0")
		defer stop()
		cfg, err := config.NewConfigFrom(map[string]interface{}{
			"server": addr,
		})
		require.NoError(t, err)
		observer := &TestObserver{}

		client := createShipperClient(t, cfg, observer)

		// Since we retry directly instead of going through a live pipeline,
		// the Publish call is synchronous and we can track state by modifying
		// local variables directly.
		retryCount := 0
		done := false
		batch := pipeline.NewBatchForTesting(
			[]publisher.Event{
				{Content: events[0]}, {Content: events[1]}, {Content: events[2]},
			},
			func(b publisher.Batch) {
				// Retry by sending directly back to Publish. In a live
				// pipeline, this would be sent through eventConsumer first
				// before calling Publish on the next free output worker.
				retryCount++
				err := client.Publish(ctx, b)
				assert.NoError(t, err, "Publish shouldn't return an error")
			},
			func() { done = true },
		)
		err = client.Publish(ctx, batch)
		assert.NoError(t, err, "Publish shouldn't return an error")

		// Only the first event is too large -- it will be retried by
		// itself and the other batch will succeed, so retryCount should
		// be 2.
		// "done" should be false because the shipper output doesn't call done
		// until upstream ingestion is confirmed via PersistedIndex.
		assert.Equal(t, 2, retryCount, "three-event batch should produce four total retries")
		assert.False(t, done, "batch should be acknowledged after Publish")

		// "batch" adds up all events passed into publish, including repeats,
		// so it should be 3 + 1 + 2 = 6
		expectedObserver := &TestObserver{split: 1, dropped: 1, batch: 6}
		require.Equal(t, expectedObserver, observer)
	})
}

// BenchmarkToShipperEvent is used to detect performance regression when the conversion function is changed.
func BenchmarkToShipperEvent(b *testing.B) {
	ts := time.Date(2022, time.July, 8, 16, 00, 00, 00, time.UTC)
	str := strings.Repeat("somelongstring", 100)

	// This event causes to go through every code path during the event conversion
	e := publisher.Event{Content: beat.Event{
		Timestamp: ts,
		Meta: mapstr.M{
			"input_id":  "someinputid",
			"stream_id": "somestreamid",
			"data_stream": mapstr.M{
				"type":      "logs",
				"namespace": "default",
				"dataset":   "default",
			},
			"number": 42,
			"string": str,
			"time":   ts,
			"bytes":  []byte(str),
			"list":   []interface{}{str, str, str},
			"nil":    nil,
		},
		Fields: mapstr.M{
			"inner": mapstr.M{
				"number": 42,
				"string": str,
				"time":   ts,
				"bytes":  []byte(str),
				"list":   []interface{}{str, str, str},
				"nil":    nil,
			},
			"number": 42,
			"string": str,
			"time":   ts,
			"bytes":  []byte(str),
			"list":   []interface{}{str, str, str},
			"nil":    nil,
		},
	}}

	for i := 0; i < b.N; i++ {
		pe, err := toShipperEvent(e)
		require.NoError(b, err)
		bytes, err := proto.Marshal(pe)
		require.NoError(b, err)
		require.NotEmpty(b, bytes)
	}
}

// runServer mocks the shipper mock server for testing
// `qSize` is a slice of the event buffer in the mock
// `err` is a preset error that the server will serve to the client
// `listenAddr` is the address for the server to listen
// returns `actualAddr` where the listener actually is and the `stop` function to stop the server
func runServer(
	t *testing.T,
	qSize int,
	errCallback func([]*messages.Event) error,
	listenAddr string,
) (actualAddr string, mock *api.ProducerMock, stop func()) {
	producer := api.NewProducerMock(qSize)
	producer.ErrorCallback = errCallback
	grpcServer := grpc.NewServer()
	pb.RegisterProducerServer(grpcServer, producer)

	listener, err := net.Listen("tcp", listenAddr)
	require.NoError(t, err)
	go func() {
		_ = grpcServer.Serve(listener)
	}()

	actualAddr = listener.Addr().String()
	stop = func() {
		grpcServer.Stop()
		listener.Close()
	}

	return actualAddr, producer, stop
}

func constErrorCallback(err error) func([]*messages.Event) error {
	return func(_ []*messages.Event) error {
		return err
	}
}

func createShipperClient(t *testing.T, cfg *config.C, observer outputs.Observer) outputs.NetworkClient {
	group, err := makeShipper(
		nil,
		beat.Info{Beat: "libbeat", IndexPrefix: "testbeat"},
		observer,
		cfg,
	)
	require.NoError(t, err)
	require.Len(t, group.Clients, 1)

	client := group.Clients[0].(outputs.NetworkClient)

	err = client.Connect()
	require.NoError(t, err)

	return client
}

func protoStruct(t *testing.T, values map[string]interface{}) *messages.Struct {
	s, err := helpers.NewStruct(values)
	require.NoError(t, err)
	return s
}

func requireEqualProto(t *testing.T, expected, actual proto.Message) {
	require.True(
		t,
		proto.Equal(expected, actual),
		fmt.Sprintf("These two protobuf messages are not equal:\nexpected: %v\nactual:  %v", expected, actual),
	)
}

// emulates the toShipperEvent, but looks for a nil meta field, and throws an error
func failMarshal(e publisher.Event) (*messages.Event, error) {
	if e.Content.Meta == nil {
		return nil, fmt.Errorf("nil meta field")
	}
	return toShipperEvent(e)
}

// mock test observer for tracking events

type TestObserver struct {
	acked     int
	dropped   int
	cancelled int
	batch     int
	duplicate int
	failed    int
	split     int

	writeError error
	readError  error

	writeBytes int
	readBytes  int

	errTooMany int
}

func (to *TestObserver) NewBatch(batch int)            { to.batch += batch }
func (to *TestObserver) Acked(acked int)               { to.acked += acked }
func (to *TestObserver) ReportLatency(_ time.Duration) {}
func (to *TestObserver) Duplicate(duplicate int)       { to.duplicate += duplicate }
func (to *TestObserver) Failed(failed int)             { to.failed += failed }
func (to *TestObserver) Dropped(dropped int)           { to.dropped += dropped }
func (to *TestObserver) Cancelled(cancelled int)       { to.cancelled += cancelled }
func (to *TestObserver) Split()                        { to.split++ }
func (to *TestObserver) WriteError(we error)           { to.writeError = we }
func (to *TestObserver) WriteBytes(wb int)             { to.writeBytes += wb }
func (to *TestObserver) ReadError(re error)            { to.readError = re }
func (to *TestObserver) ReadBytes(rb int)              { to.readBytes += rb }
func (to *TestObserver) ErrTooMany(err int)            { to.errTooMany = +err }
