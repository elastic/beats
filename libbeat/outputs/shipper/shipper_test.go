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
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
	"github.com/elastic/beats/v7/libbeat/outputs/shipper/api"
	"github.com/elastic/beats/v7/libbeat/publisher"
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
	//logp.DevelopmentSetup()
	events := []beat.Event{
		{
			Timestamp: time.Now(),
			Meta:      mapstr.M{"event": "first"},
			Fields:    mapstr.M{"a": "b"},
		},
		{
			Timestamp: time.Now(),
			Meta:      nil, // see failMarshal()
			Fields:    mapstr.M{"a": "b"},
		},
		{
			Timestamp: time.Now(),
			Meta:      mapstr.M{"event": "third"},
			Fields:    mapstr.M{"e": "f"},
		},
	}

	cases := []struct {
		name        string
		events      []beat.Event
		expSignals  []outest.BatchSignal
		serverError error
		expError    string
		// note: this sets the queue size used by the mock output
		// if the mock shipper recieves more than this count of events, the test will fail
		qSize            int
		observerExpected *TestObserver
		marshalMethod    func(e publisher.Event) (*messages.Event, error)
	}{
		{
			name:          "sends a batch",
			events:        events,
			marshalMethod: toShipperEvent,
			expSignals: []outest.BatchSignal{
				{
					Tag: outest.BatchACK,
				},
			},
			qSize:            3,
			observerExpected: &TestObserver{batch: 3, acked: 3},
		},
		{
			name:   "retries not accepted events",
			events: events,
			expSignals: []outest.BatchSignal{
				{
					Tag: outest.BatchACK,
				},
			},
			marshalMethod:    failMarshal, // emulate a dropped event
			qSize:            3,
			observerExpected: &TestObserver{batch: 3, dropped: 1, acked: 2},
		},
		{
			name:   "cancels the batch if server error",
			events: events,
			expSignals: []outest.BatchSignal{
				{
					Tag: outest.BatchCancelled,
				},
			},
			marshalMethod:    toShipperEvent,
			qSize:            3,
			observerExpected: &TestObserver{cancelled: 3, batch: 3},
			serverError:      errors.New("some error"),
			expError:         "failed to publish the batch to the shipper, none of the 3 events were accepted",
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

			addr, producer, stop := runServer(t, tc.qSize, tc.serverError, "localhost:0")
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
func runServer(t *testing.T, qSize int, err error, listenAddr string) (actualAddr string, mock *api.ProducerMock, stop func()) {
	producer := api.NewProducerMock(qSize)
	producer.Error = err
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
func protoStructValue(t *testing.T, values map[string]interface{}) *messages.Value {
	s := protoStruct(t, values)
	return helpers.NewStructValue(s)
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

	writeError error
	readError  error

	writeBytes int
	readBytes  int

	errTooMany int
}

func (to *TestObserver) NewBatch(batch int)      { to.batch += batch }
func (to *TestObserver) Acked(acked int)         { to.acked += acked }
func (to *TestObserver) Duplicate(duplicate int) { to.duplicate += duplicate }
func (to *TestObserver) Failed(failed int)       { to.failed += failed }
func (to *TestObserver) Dropped(dropped int)     { to.dropped += dropped }
func (to *TestObserver) Cancelled(cancelled int) { to.cancelled += cancelled }
func (to *TestObserver) WriteError(we error)     { to.writeError = we }
func (to *TestObserver) WriteBytes(wb int)       { to.writeBytes += wb }
func (to *TestObserver) ReadError(re error)      { to.readError = re }
func (to *TestObserver) ReadBytes(rb int)        { to.readBytes += rb }
func (to *TestObserver) ErrTooMany(err int)      { to.errTooMany = +err }
