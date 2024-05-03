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
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/memqueue"
	"github.com/elastic/beats/v7/libbeat/tests/resources"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func makePipeline(t *testing.T, settings Settings, qu queue.Queue) *Pipeline {
	p, err := New(beat.Info{},
		Monitors{},
		conf.Namespace{},
		outputs.Group{},
		settings,
	)
	require.NoError(t, err)
	// Inject a test queue so the outputController doesn't create one
	p.outputController.queue = qu

	return p
}

func TestClient(t *testing.T) {
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

		logp.TestingSetup()

		for name, test := range cases {
			t.Run(name, func(t *testing.T) {
				routinesChecker := resources.NewGoroutinesChecker()
				defer routinesChecker.Check(t)

				pipeline := makePipeline(t, Settings{}, makeTestQueue())
				defer pipeline.Close()

				client, err := pipeline.ConnectWith(beat.ClientConfig{})
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

				test.close(client, func() {
					client.Close()
				})
				wg.Wait()
			})
		}
	})

	t.Run("no infinite loop when processing fails", func(t *testing.T) {
		logp.TestingSetup()
		l := logp.L()

		// a small in-memory queue with a very short flush interval
		q := memqueue.NewQueue(l, nil, memqueue.Settings{
			Events:        5,
			MaxGetRequest: 1,
			FlushTimeout:  time.Millisecond,
		}, 5, nil)

		// model a processor that we're going to make produce errors after
		p := &testProcessor{}
		ps := testProcessorSupporter{Processor: p}

		// now we create a pipeline that makes sure that all
		// events are acked while shutting down
		pipeline := makePipeline(t, Settings{
			WaitClose:     100 * time.Millisecond,
			WaitCloseMode: WaitOnPipelineClose,
			Processors:    ps,
		}, q)
		client, err := pipeline.Connect()
		require.NoError(t, err)
		defer client.Close()

		// consuming all the published events
		var received []beat.Event
		done := make(chan struct{})
		go func() {
			for {
				batch, err := q.Get(2, 0)
				if errors.Is(err, io.EOF) {
					break
				}
				assert.NoError(t, err)
				if batch == nil {
					continue
				}
				for i := 0; i < batch.Count(); i++ {
					e := batch.Entry(i).(publisher.Event)
					received = append(received, e.Content)
				}
				batch.Done()
			}
			close(done)
		}()

		sent := []beat.Event{
			{
				Fields: mapstr.M{"number": 1},
			},
			{
				Fields: mapstr.M{"number": 2},
			},
			{
				Fields: mapstr.M{"number": 3},
			},
			{
				Fields: mapstr.M{"number": 4},
			},
		}

		expected := []beat.Event{
			{
				Fields: mapstr.M{"number": 1, "test": "value"},
			},
			{
				Fields: mapstr.M{"number": 2, "test": "value"},
			},
			// {
			// 	// this event must be excluded due to the processor error
			// 	Fields: mapstr.M{"number": 3},
			// },
			{
				Fields: mapstr.M{"number": 4, "test": "value"},
			},
		}

		client.PublishAll(sent[:2]) // first 2

		// this causes our processor to malfunction and produce errors for all events
		p.ErrorSwitch()

		client.PublishAll(sent[2:3]) // number 3

		// back to normal
		p.ErrorSwitch()

		client.PublishAll(sent[3:]) // number 4

		client.Close()
		pipeline.Close()

		// waiting for all events to be consumed from the queue
		<-done
		require.Equal(t, expected, received)
	})
}

func TestClientWaitClose(t *testing.T) {
	routinesChecker := resources.NewGoroutinesChecker()
	defer routinesChecker.Check(t)

	makePipeline := func(settings Settings, qu queue.Queue) *Pipeline {
		p, err := New(beat.Info{},
			Monitors{},
			conf.Namespace{},
			outputs.Group{},
			settings,
		)
		if err != nil {
			panic(err)
		}
		// Inject a test queue so the outputController doesn't create one
		p.outputController.queue = qu

		return p
	}
	logp.TestingSetup()

	q := memqueue.NewQueue(logp.L(), nil, memqueue.Settings{Events: 1}, 0, nil)
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
		output := newMockClient(func(batch publisher.Batch) error {
			batch.ACK()
			return nil
		})
		defer output.Close()
		pipeline.outputController.Set(outputs.Group{Clients: []outputs.Client{output}})
		defer pipeline.outputController.Set(outputs.Group{})

		client.Publish(beat.Event{})

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

type testProcessor struct{ error bool }

func (p *testProcessor) String() string {
	return "testProcessor"
}
func (p *testProcessor) Run(in *beat.Event) (event *beat.Event, err error) {
	if p.error {
		return nil, errors.New("test error")
	}
	_, err = in.Fields.Put("test", "value")
	return in, err
}

func (p *testProcessor) ErrorSwitch() {
	p.error = !p.error
}

type testProcessorSupporter struct {
	beat.Processor
}

// Create a running processor interface based on the given config
func (p testProcessorSupporter) Create(cfg beat.ProcessingConfig, drop bool) (beat.Processor, error) {
	return p.Processor, nil
}

// Processors returns a list of config strings for the given processor, for debug purposes
func (p testProcessorSupporter) Processors() []string {
	return []string{p.Processor.String()}
}

// Close the processor supporter
func (p testProcessorSupporter) Close() error {
	return processors.Close(p.Processor)
}
