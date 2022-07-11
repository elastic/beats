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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
	sc "github.com/elastic/beats/v7/libbeat/outputs/shipper/api"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestToShipperEvent(t *testing.T) {
	ts := time.Now().Truncate(time.Second)

	cases := []struct {
		name   string
		value  publisher.Event
		exp    *sc.Event
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
			exp: &sc.Event{
				Timestamp:  timestamppb.New(ts),
				Source:     &sc.Source{},
				DataStream: &sc.DataStream{},
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
			exp: &sc.Event{
				Timestamp: timestamppb.New(ts),
				Source: &sc.Source{
					InputId:  "input",
					StreamId: "stream",
				},
				DataStream: &sc.DataStream{
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
		{
			name: "returns error if failed to convert metadata",
			value: publisher.Event{
				Content: beat.Event{
					Timestamp: ts,
					Meta: mapstr.M{
						"metafield": ts, // timestamp is a wrong type
					},
				},
			},
			expErr: "failed to convert event metadata",
		},
		{
			name: "returns error if failed to convert fields",
			value: publisher.Event{
				Content: beat.Event{
					Timestamp: ts,
					Fields: mapstr.M{
						"field": ts, // timestamp is a wrong type
					},
				},
			},
			expErr: "failed to convert event fields",
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

func TestConvertMapStr(t *testing.T) {
	cases := []struct {
		name   string
		value  mapstr.M
		exp    *structpb.Value
		expErr string
	}{
		{
			name: "nil returns nil",
			exp:  structpb.NewNullValue(),
		},
		{
			name:  "empty map returns empty struct",
			value: mapstr.M{},
			exp:   protoStructValue(t, nil),
		},
		{
			name: "returns error when type is not supported",
			value: mapstr.M{
				"key": time.Now(),
			},
			expErr: "invalid type: time.Time",
		},
		{
			name: "values are preserved",
			value: mapstr.M{
				"key1": "string",
				"key2": 42,
				"key3": 42.2,
				"key4": mapstr.M{
					"subkey1": "string",
					"subkey2": mapstr.M{
						"subsubkey1": "string",
					},
				},
			},
			exp: protoStructValue(t, map[string]interface{}{
				"key1": "string",
				"key2": 42,
				"key3": 42.2,
				"key4": map[string]interface{}{
					"subkey1": "string",
					"subkey2": map[string]interface{}{
						"subsubkey1": "string",
					},
				},
			}),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			converted, err := convertMapStr(tc.value)
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
			Timestamp: time.Now(),
			Meta:      mapstr.M{"event": "first"},
			Fields:    mapstr.M{"a": "b"},
		},
		{
			Timestamp: time.Now(),
			Meta:      mapstr.M{"event": "second", "dropped": true, "invalid": struct{}{}}, // this event is always dropped
			Fields:    mapstr.M{"c": "d"},
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
		qSize       int
	}{
		{
			name:   "sends a batch excluding dropped",
			events: events[:1],
			expSignals: []outest.BatchSignal{
				{
					Tag: outest.BatchACK,
				},
			},
			qSize: 2,
		},
		{
			name:   "retries not accepted events",
			events: events,
			expSignals: []outest.BatchSignal{
				{
					Tag:    outest.BatchRetryEvents,
					Events: toPublisherEvents(events[2:]),
				},
			},
			qSize: 1,
		},
		{
			name:   "cancels the batch if server error",
			events: events,
			expSignals: []outest.BatchSignal{
				{
					Tag: outest.BatchCancelled,
				},
			},
			qSize:       3,
			serverError: errors.New("some error"),
			expError:    "failed to publish the batch to the shipper, none of the 2 events were accepted",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			addr, stop := runServer(t, tc.qSize, tc.serverError, "localhost:0")
			defer stop()

			cfg, err := config.NewConfigFrom(map[string]interface{}{
				"server": addr,
			})
			require.NoError(t, err)

			group, err := makeShipper(
				nil,
				beat.Info{Beat: "libbeat", IndexPrefix: "testbeat"},
				outputs.NewNilObserver(),
				cfg,
			)
			require.NoError(t, err)
			require.Len(t, group.Clients, 1)

			batch := outest.NewBatch(tc.events...)

			err = group.Clients[0].(outputs.Connectable).Connect()
			require.NoError(t, err)

			err = group.Clients[0].Publish(ctx, batch)
			if tc.expError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expError)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tc.expSignals, batch.Signals)
		})
	}

	t.Run("cancel the batch when the server is not available", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		addr, stop := runServer(t, 5, nil, "localhost:0")
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

		group, err := makeShipper(
			nil,
			beat.Info{Beat: "libbeat", IndexPrefix: "testbeat"},
			outputs.NewNilObserver(),
			cfg,
		)
		require.NoError(t, err)
		require.Len(t, group.Clients, 1)

		client := group.Clients[0].(outputs.NetworkClient)

		err = client.Connect()
		require.NoError(t, err)

		// Should successfully publish with the server running
		batch := outest.NewBatch(events...)
		err = client.Publish(ctx, batch)
		require.NoError(t, err)
		expSignals := []outest.BatchSignal{
			{
				Tag: outest.BatchACK,
			},
		}
		require.Equal(t, expSignals, batch.Signals)

		stop() // now stop the server and try sending again

		batch = outest.NewBatch(events...) // resetting the batch signals
		err = client.Publish(ctx, batch)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to publish the batch to the shipper, none of the 2 events were accepted")
		expSignals = []outest.BatchSignal{
			{
				Tag: outest.BatchCancelled, // "cancelled" means there will be a retry without decreasing the TTL
			},
		}
		require.Equal(t, expSignals, batch.Signals)
		client.Close()

		// Start the server again
		_, stop = runServer(t, 5, nil, addr)
		defer stop()

		batch = outest.NewBatch(events...) // resetting the signals
		expSignals = []outest.BatchSignal{
			{
				Tag: outest.BatchACK,
			},
		}

		// The backoff wrapper should take care of the errors and
		// retries while the server is still starting
		err = client.Connect()
		require.NoError(t, err)

		err = client.Publish(ctx, batch)
		require.NoError(t, err)
		require.Equal(t, expSignals, batch.Signals)
	})
}

// runServer mocks the shipper mock server for testing
// `qSize` is a slice of the event buffer in the mock
// `err` is a preset error that the server will serve to the client
// `listenAddr` is the address for the server to listen
// returns `actualAddr` where the listener actually is and the `stop` function to stop the server
func runServer(t *testing.T, qSize int, err error, listenAddr string) (actualAddr string, stop func()) {
	producer := sc.NewProducerMock(qSize)
	producer.Error = err
	grpcServer := grpc.NewServer()
	sc.RegisterProducerServer(grpcServer, producer)

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

	return actualAddr, stop
}

func protoStruct(t *testing.T, values map[string]interface{}) *structpb.Struct {
	s, err := structpb.NewStruct(values)
	require.NoError(t, err)
	return s
}
func protoStructValue(t *testing.T, values map[string]interface{}) *structpb.Value {
	s := protoStruct(t, values)
	return structpb.NewStructValue(s)
}

func requireEqualProto(t *testing.T, expected, actual proto.Message) {
	require.True(
		t,
		proto.Equal(expected, actual),
		fmt.Sprintf("These two protobuf messages are not equal:\nexpected: %v\nactual:  %v", expected, actual),
	)
}

func toPublisherEvents(events []beat.Event) []publisher.Event {
	converted := make([]publisher.Event, 0, len(events))
	for _, e := range events {
		converted = append(converted, publisher.Event{Content: e})
	}
	return converted
}
