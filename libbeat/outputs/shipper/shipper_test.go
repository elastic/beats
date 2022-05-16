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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
	sc "github.com/elastic/beats/v7/libbeat/outputs/shipper/api"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

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
			exp:   protoStruct(t, nil),
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
			exp: protoStruct(t, map[string]interface{}{
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
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			producer := sc.NewProducerMock(tc.qSize)
			producer.Error = tc.serverError
			grpcServer := grpc.NewServer()
			sc.RegisterProducerServer(grpcServer, producer)

			listener, err := net.Listen("tcp", "localhost:0") // random available port
			require.NoError(t, err)

			defer grpcServer.Stop()
			go func() {
				_ = grpcServer.Serve(listener)
			}()

			cfg, err := config.NewConfigFrom(map[string]interface{}{
				"server": listener.Addr().String(),
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
			err = group.Clients[0].Publish(ctx, batch)
			require.NoError(t, err)

			require.Equal(t, tc.expSignals, batch.Signals)
		})
	}

	t.Run("cancel the batch when the server is not available", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		producer := sc.NewProducerMock(5)
		grpcServer := grpc.NewServer()
		sc.RegisterProducerServer(grpcServer, producer)

		listener, err := net.Listen("tcp", "localhost:0") // random available port
		require.NoError(t, err)
		defer grpcServer.Stop()

		cfg, err := config.NewConfigFrom(map[string]interface{}{
			"server":  listener.Addr().String(),
			"timeout": 1, // 1 sec
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

		batch := outest.NewBatch(events...)

		// try to publish without the server running
		err = group.Clients[0].Publish(ctx, batch)
		require.NoError(t, err)

		expSignals := []outest.BatchSignal{
			{
				Tag: outest.BatchCancelled, // "cancelled" means there will be a retry without decreasing the TTL
			},
		}
		require.Equal(t, expSignals, batch.Signals)

		// Start the server
		go func() {
			_ = grpcServer.Serve(listener)
		}()

		var actSignals []outest.BatchSignal
		expSignals = []outest.BatchSignal{
			{
				Tag: outest.BatchACK,
			},
		}

		// Poll for the batch to be acknowledged. The gRPC server takes a variable amount
		// of time to start, so some retries are necessary.
		require.Eventually(t, func() bool {
			batch = outest.NewBatch(events...)
			err = group.Clients[0].Publish(ctx, batch)
			require.NoError(t, err)

			actSignals = batch.Signals
			return reflect.DeepEqual(expSignals, batch.Signals)
		}, 5*time.Second, 500*time.Millisecond)

		// Use require.Equal() on the final signal set. If the Eventually() loop above
		// failed this will print the difference between the signal sets.
		require.Equal(t, expSignals, actSignals)
	})
}

func protoStruct(t *testing.T, values map[string]interface{}) *structpb.Value {
	s, err := structpb.NewStruct(values)
	require.NoError(t, err)
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
