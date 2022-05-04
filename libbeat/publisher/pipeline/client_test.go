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

package pipeline

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/memqueue"
	"github.com/elastic/beats/v7/libbeat/tests/resources"
	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestClient(t *testing.T) {
	makePipeline := func(settings Settings, qu queue.Queue) *Pipeline {
		p, err := New(beat.Info{},
			Monitors{},
			func(_ queue.ACKListener) (queue.Queue, error) {
				return qu, nil
			},
			outputs.Group{},
			settings,
		)
		if err != nil {
			panic(err)
		}

		return p
	}

	t.Run("client close", func(t *testing.T) {
		// Note: no asserts. If closing fails we have a deadlock, because Publish
		// would block forever

		cases := map[string]struct {
			context bool
			close   func(client beat.Client, cancel func())
		}{
			"close unblocks client without context": {
				context: false,
				close: func(client beat.Client, _ func()) {
					client.Close()
				},
			},
			"close unblocks client with context": {
				context: true,
				close: func(client beat.Client, _ func()) {
					client.Close()
				},
			},
			"context cancel unblocks client": {
				context: true,
				close: func(client beat.Client, cancel func()) {
					cancel()
				},
			},
		}

		if testing.Verbose() {
			logp.TestingSetup()
		}

		for name, test := range cases {
			t.Run(name, func(t *testing.T) {
				routinesChecker := resources.NewGoroutinesChecker()
				defer routinesChecker.Check(t)

				pipeline := makePipeline(Settings{}, makeBlockingQueue())
				defer pipeline.Close()

				var ctx context.Context
				var cancel func()
				if test.context {
					ctx, cancel = context.WithCancel(context.Background())
				}

				client, err := pipeline.ConnectWith(beat.ClientConfig{
					CloseRef: ctx,
				})
				if err != nil {
					t.Fatal(err)
				}
				defer client.Close()

				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					defer wg.Done()
					client.Publish(beat.Event{})
				}()

				test.close(client, cancel)
				wg.Wait()
			})
		}
	})
}

func TestClientWaitClose(t *testing.T) {
	routinesChecker := resources.NewGoroutinesChecker()
	defer routinesChecker.Check(t)

	makePipeline := func(settings Settings, qu queue.Queue) *Pipeline {
		p, err := New(beat.Info{},
			Monitors{},
			func(queue.ACKListener) (queue.Queue, error) { return qu, nil },
			outputs.Group{},
			settings,
		)
		if err != nil {
			panic(err)
		}

		return p
	}
	if testing.Verbose() {
		logp.TestingSetup()
	}

	q := memqueue.NewQueue(logp.L(), memqueue.Settings{Events: 1})
	pipeline := makePipeline(Settings{}, q)
	defer pipeline.Close()

	t.Run("WaitClose blocks", func(t *testing.T) {
		client, err := pipeline.ConnectWith(beat.ClientConfig{
			WaitClose: 500 * time.Millisecond,
		})
		if err != nil {
			t.Fatal(err)
		}
		defer client.Close()

		// Send an event which never gets acknowledged.
		client.Publish(beat.Event{})

		closed := make(chan struct{})
		go func() {
			defer close(closed)
			client.Close()
		}()

		select {
		case <-closed:
			t.Fatal("expected Close to wait for event acknowledgement")
		case <-time.After(100 * time.Millisecond):
		}

		select {
		case <-closed:
		case <-time.After(10 * time.Second):
			t.Fatal("expected Close to stop waiting after WaitClose elapses")
		}
	})

	t.Run("ACKing events unblocks WaitClose", func(t *testing.T) {
		client, err := pipeline.ConnectWith(beat.ClientConfig{
			WaitClose: time.Minute,
		})
		if err != nil {
			t.Fatal(err)
		}
		defer client.Close()

		// Send an event which gets acknowledged immediately.
		client.Publish(beat.Event{})
		output := newMockClient(func(batch publisher.Batch) error {
			batch.ACK()
			return nil
		})
		defer output.Close()
		pipeline.output.Set(outputs.Group{Clients: []outputs.Client{output}})
		defer pipeline.output.Set(outputs.Group{})

		closed := make(chan struct{})
		go func() {
			defer close(closed)
			client.Close()
		}()

		select {
		case <-closed:
		case <-time.After(10 * time.Second):
			t.Fatal("expected Close to stop waiting after event acknowledgement")
		}
	})
}

func TestMonitoring(t *testing.T) {
	const (
		maxEvents  = 123
		batchSize  = 456
		numClients = 42
	)
	var config Config
	err := conf.MustNewConfigFrom(map[string]interface{}{
		"queue.mem.events":           maxEvents,
		"queue.mem.flush.min_events": 1,
	}).Unpack(&config)
	require.NoError(t, err)

	metrics := monitoring.NewRegistry()
	telemetry := monitoring.NewRegistry()
	pipeline, err := Load(
		beat.Info{},
		Monitors{
			Metrics:   metrics,
			Telemetry: telemetry,
		},
		config,
		processing.Supporter(nil),
		func(outputs.Observer) (string, outputs.Group, error) {
			clients := make([]outputs.Client, numClients)
			for i := range clients {
				clients[i] = newMockClient(func(publisher.Batch) error {
					return nil
				})
			}
			return "output_name", outputs.Group{
				BatchSize: batchSize,
				Clients:   clients,
			}, nil
		},
	)
	require.NoError(t, err)
	defer pipeline.Close()

	metricsSnapshot := monitoring.CollectFlatSnapshot(metrics, monitoring.Full, true)
	assert.Equal(t, int64(maxEvents), metricsSnapshot.Ints["pipeline.queue.max_events"])

	telemetrySnapshot := monitoring.CollectFlatSnapshot(telemetry, monitoring.Full, true)
	assert.Equal(t, "output_name", telemetrySnapshot.Strings["output.name"])
	assert.Equal(t, int64(batchSize), telemetrySnapshot.Ints["output.batch_size"])
	assert.Equal(t, int64(numClients), telemetrySnapshot.Ints["output.clients"])
}
