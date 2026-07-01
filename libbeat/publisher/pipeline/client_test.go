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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/memqueue"
	"github.com/elastic/beats/v7/libbeat/tests/resources"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func makePipeline(t *testing.T, settings Settings, qu queue.Queue[publisher.Event]) *Pipeline {
	t.Helper()
	logger := logptest.NewTestingLogger(t, "")
	p, err := New(beat.Info{Logger: logger},
		Monitors{},
		conf.Namespace{},
		outputs.Group{},
		settings,
	)
	require.NoError(t, err)
	if outputController, ok := p.outputController.(*processOutputController); ok {
		// Inject a test queue so the outputController doesn't create one
		outputController.queue = qu
	}

	return p
}

func TestClient(t *testing.T) {
	t.Run("client close", func(t *testing.T) {
		// Note: no asserts. If closing fails we have a deadlock, because Publish
		// would block forever

		routinesChecker := resources.NewGoroutinesChecker()
		defer routinesChecker.Check(t)

		pipeline := makePipeline(t, Settings{}, makeTestQueue())
		defer func() { _ = pipeline.Disconnect(t.Context()) }()

		client, err := pipeline.ConnectWith(beat.ClientConfig{})
		if err != nil {
			t.Fatal(err)
		}

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			client.Publish(beat.Event{})
		}()

		client.Close()
		wg.Wait()
	})

	t.Run("no infinite loop when processing fails", func(t *testing.T) {
		l := logptest.NewTestingLogger(t, "")

		// a small in-memory queue with a very short flush interval
		q := memqueue.NewQueue[publisher.Event](l, nil, memqueue.Settings{
			Events:        5,
			MaxGetRequest: 1,
			FlushTimeout:  time.Millisecond,
		}, 5, nil)

		// model a processor that we're going to make produce errors after
		processorAddField := func(in *beat.Event) (event *beat.Event, err error) {
			_, err = in.Fields.Put("test", "value")
			return in, err
		}
		processorsErr := func(in *beat.Event) (event *beat.Event, err error) {
			return nil, errors.New("test error")
		}
		p := &testProcessor{processorFn: processorAddField}
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
				batch, err := q.Get(2)
				if errors.Is(err, io.EOF) {
					break
				}
				assert.NoError(t, err)
				if batch == nil {
					continue
				}
				for i := 0; i < batch.Count(); i++ {
					e := batch.Entry(i)
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
		p.processorFn = processorsErr

		client.PublishAll(sent[2:3]) // number 3

		// back to normal
		p.processorFn = processorAddField

		client.PublishAll(sent[3:]) // number 4

		require.NoError(t, client.Close(), "failed closing pipeline client")
		require.NoError(t, pipeline.Disconnect(t.Context()), "failed closing pipeline")

		// waiting for all events to be consumed from the queue
		<-done
		require.Equal(t, expected, received)
	})
}

// TestDisconnectIsIdempotent verifies that the second stage of client shutdown
// runs its finalization exactly once, even if disconnect is called more than
// once (e.g. by both a per-client path and the Pipeline).
func TestDisconnectIsIdempotent(t *testing.T) {
	removed := 0
	c := &client{
		logger:         logp.NewNopLogger(),
		observer:       nilObserver,
		eventListener:  acker.Nil(),
		clientListener: &mockClientListener{},
		onRemove:       func() { removed++ },
	}

	c.disconnect()
	c.disconnect() // must hit the idempotency guard and do nothing

	assert.Equal(t, 1, removed, "disconnect must finalize (and unregister) exactly once")
}

// TestClientFinalizedWhenDrainedMidRun verifies that a client closed while the
// pipeline keeps running is finalized (stage two) by the reaper as soon as its
// events drain — it is unregistered without waiting for a pipeline disconnect.
func TestClientFinalizedWhenDrainedMidRun(t *testing.T) {
	routinesChecker := resources.NewGoroutinesChecker()
	defer routinesChecker.Check(t)

	pipeline := makePipeline(t, Settings{}, makeTestQueue())
	defer func() { _ = pipeline.Disconnect(t.Context()) }()

	client, err := pipeline.ConnectWith(beat.ClientConfig{})
	require.NoError(t, err)

	pipeline.clientsMu.Lock()
	require.Len(t, pipeline.clients, 1, "client should be registered after connect")
	pipeline.clientsMu.Unlock()

	// Close mid-run; the pipeline is NOT disconnected. The test producer's
	// ack-wait channel is already closed, so the reaper should finalize the
	// client and unregister it promptly.
	require.NoError(t, client.Close())

	require.Eventually(t, func() bool {
		pipeline.clientsMu.Lock()
		defer pipeline.clientsMu.Unlock()
		return len(pipeline.clients) == 0
	}, 5*time.Second, 5*time.Millisecond,
		"reaper should finalize and unregister a drained client without a pipeline disconnect")
}

// TestReaperFinalizesClientThatDrainsAfterClose verifies that a client whose
// events are still in flight at Close stays registered while the reaper
// re-polls, and is finalized once its events drain (its ACKWaitChan closes).
// This exercises the reaper's re-poll path for not-yet-drained clients.
func TestReaperFinalizesClientThatDrainsAfterClose(t *testing.T) {
	routinesChecker := resources.NewGoroutinesChecker()
	defer routinesChecker.Check(t)

	// A shared ack-wait channel that the test holds open until it decides the
	// clients' events have "drained".
	ackWait := make(chan struct{})
	tq := &testQueue{
		producer: func(_ queue.ProducerConfig) queue.Producer[publisher.Event] {
			return &testProducer{
				publish: func(bool, publisher.Event) (queue.EntryID, bool) { return 1, true },
				ackWait: ackWait,
			}
		},
		done: make(chan struct{}),
	}
	pipeline := makePipeline(t, Settings{}, tq)
	defer func() { _ = pipeline.Disconnect(t.Context()) }()

	// Close one client; it is handed to the reaper but cannot drain yet (the
	// test holds ackWait open). Close a second while the first is still pending
	// so the reaper also sees it via the notify path.
	c1, err := pipeline.ConnectWith(beat.ClientConfig{})
	require.NoError(t, err)
	require.NoError(t, c1.Close())

	c2, err := pipeline.ConnectWith(beat.ClientConfig{})
	require.NoError(t, err)
	require.NoError(t, c2.Close())

	// Across several reaper ticks both clients must stay registered: their
	// events have not drained, so the reaper's re-poll must not finalize them.
	require.Never(t, func() bool {
		pipeline.clientsMu.Lock()
		defer pipeline.clientsMu.Unlock()
		return len(pipeline.clients) != 2
	}, 3*reaperInterval, reaperInterval/2,
		"clients must stay registered until their events drain")

	// Drain: now both clients' events are acknowledged.
	close(ackWait)
	require.Eventually(t, func() bool {
		pipeline.clientsMu.Lock()
		defer pipeline.clientsMu.Unlock()
		return len(pipeline.clients) == 0
	}, 5*time.Second, 5*time.Millisecond,
		"reaper must finalize both clients once their events drain")
}

// TestEmptyProducerACKWaitChanClosed verifies the placeholder producer used
// when publishing is disabled reports an already-closed ack-wait channel, so a
// caller selecting on it never blocks.
func TestEmptyProducerACKWaitChanClosed(t *testing.T) {
	select {
	case <-emptyProducer{}.ACKWaitChan():
	default:
		t.Fatal("emptyProducer.ACKWaitChan must be closed")
	}
}

// TestCloseSerializesWithInFlightPublish verifies that Close serializes with an
// in-flight Publish: while a Publish holds the client mutex (here, blocked
// inside a processor), Close must not proceed to flip isOpen / close the
// producer until that Publish completes. This is the mutex-ordering guarantee
// from the fix for https://github.com/elastic/beats/issues/49390. In the strict
// two-stage model Close no longer waits for acknowledgments, so once the
// in-flight Publish finishes Close returns promptly.
func TestCloseSerializesWithInFlightPublish(t *testing.T) {
	// inProcessor is closed once the processor is running, letting the
	// test know Publish is in the race window.
	inProcessor := make(chan struct{})
	// releaseProcessor is closed by the test to let the processor finish.
	releaseProcessor := make(chan struct{})

	c := &client{
		logger: logp.NewNopLogger(),
		processors: &testProcessor{
			processorFn: func(in *beat.Event) (*beat.Event, error) {
				close(inProcessor) // signal: we are inside the processor
				<-releaseProcessor // block until test says go
				return in, nil
			},
		},
		producer: &testProducer{
			publish: func(_ bool, event publisher.Event) (queue.EntryID, bool) {
				return 1, true
			},
		},
		observer:       nilObserver,
		eventListener:  acker.Nil(),
		clientListener: &mockClientListener{},
	}
	c.isOpen.Store(true)

	// Start Publish in a goroutine — it will block inside the processor while
	// holding the client mutex.
	publishDone := make(chan struct{})
	go func() {
		defer close(publishDone)
		c.Publish(beat.Event{Fields: mapstr.M{"hello": "world"}})
	}()

	// Wait until Publish is inside the processor (holding the mutex).
	<-inProcessor

	closeDone := make(chan struct{})
	go func() {
		defer close(closeDone)
		c.Close()
	}()

	// Close must block on the client mutex until the in-flight Publish finishes.
	select {
	case <-closeDone:
		t.Fatal("Close returned while an in-flight Publish still held the client mutex")
	case <-time.After(100 * time.Millisecond):
		// Good — Close is serialized behind the in-flight Publish.
	}

	// Release the processor so Publish completes and releases the mutex.
	close(releaseProcessor)
	<-publishDone

	// Close should now return promptly: strict mode does not wait for acks.
	select {
	case <-closeDone:
		// Good — Close returned once the in-flight Publish completed.
	case <-time.After(10 * time.Second):
		t.Fatal("Close did not return after the in-flight Publish completed")
	}
}

func TestClientWaitClose(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	q := memqueue.NewQueue[publisher.Event](logger, nil, memqueue.Settings{Events: 1}, 0, nil)
	pipeline := makePipeline(t, Settings{}, q)
	defer func() { _ = pipeline.Disconnect(t.Context()) }()

	// In the strict two-stage model (issue #50104) client.Close no longer blocks
	// for ClientConfig.WaitClose: it stops new events, closes the producer, and
	// returns immediately. Waiting for acknowledgments is now the pipeline's
	// responsibility at Disconnect time.
	t.Run("Close returns immediately without waiting for acks", func(t *testing.T) {
		routinesChecker := resources.NewGoroutinesChecker()
		defer routinesChecker.Check(t)

		client, err := pipeline.ConnectWith(beat.ClientConfig{
			WaitClose: time.Minute,
		})
		if err != nil {
			t.Fatal(err)
		}

		// Send an event which never gets acknowledged (no output is configured).
		client.Publish(beat.Event{})

		closed := make(chan struct{})
		go func() {
			defer close(closed)
			client.Close()
		}()

		// Despite WaitClose being a minute and the event never being acked,
		// Close must return promptly.
		select {
		case <-closed:
		case <-time.After(10 * time.Second):
			t.Fatal("Close must return immediately in the strict two-stage model, even with unacknowledged events")
		}
	})
}

func TestMonitoring(t *testing.T) {
	t.Run("output metrics", func(t *testing.T) {
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
		beatInfo := beat.Info{Logger: logptest.NewTestingLogger(t, "")}
		pipeline, err := Load(
			beatInfo,
			Monitors{
				Metrics:   metrics,
				Telemetry: telemetry,
				Logger:    logp.NewNopLogger(),
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
		defer func() { _ = pipeline.Disconnect(t.Context()) }()

		telemetrySnapshot := monitoring.CollectFlatSnapshot(telemetry, monitoring.Full, true)
		assert.Equal(t, "output_name", telemetrySnapshot.Strings["output.name"])
		assert.Equal(t, int64(batchSize), telemetrySnapshot.Ints["output.batch_size"])
		assert.Equal(t, int64(numClients), telemetrySnapshot.Ints["output.clients"])
	})

	t.Run("input metrics", func(t *testing.T) {
		testInputMetrics(t,
			beat.Info{},
			beat.ClientConfig{ClientListener: &mockClientListener{}})
	})

	t.Run("no input metrics - nil ClientConfig", func(t *testing.T) {
		testInputMetrics(
			t, beat.Info{}, beat.ClientConfig{})
	})
}

func testInputMetrics(t *testing.T, beatInfo beat.Info, clientCfg beat.ClientConfig) {

	var config Config
	err := conf.MustNewConfigFrom(map[string]interface{}{
		"queue.mem.events":           32,
		"queue.mem.flush.min_events": 1,
		"queue.mem.flush.timeout":    time.Millisecond,
	}).Unpack(&config)
	require.NoError(t, err, "failed creating config")

	filterMeKey := "filter_me"

	metrics := monitoring.NewRegistry()
	telemetry := monitoring.NewRegistry()
	logger := logptest.NewTestingLogger(t, "")
	pipeline, err := Load(
		beat.Info{
			Logger: logger,
		},
		Monitors{
			Metrics:   metrics,
			Telemetry: telemetry,
			Logger:    logger,
		},
		config,
		testProcessorSupporter{
			Processor: processorList{
				processors: []beat.Processor{
					&testProcessor{
						name: "filterProcessor",
						processorFn: func(in *beat.Event) (*beat.Event, error) {
							rawFilterMe, err := in.Fields.GetValue(filterMeKey)
							if err != nil && !errors.Is(err, mapstr.ErrKeyNotFound) {
								require.NoError(t, err, "could not get filter_me from Fields")
							}

							filterMe, ok := rawFilterMe.(bool)
							if filterMe && ok {
								return nil, nil
							}
							return in, nil
						},
					},
				},
			},
		},
		func(outputs.Observer) (string, outputs.Group, error) {
			return "output_name", outputs.Group{Clients: []outputs.Client{
				newMockClient(func(publisher.Batch) error { return nil })},
			}, nil
		},
	)
	require.NoError(t, err)

	c, err := pipeline.ConnectWith(clientCfg)
	require.NoError(t, err, "pipeline.ConnectWith failed")

	cc, ok := c.(*client)
	require.True(t, ok, "pipeline.ConnectWith return value cannot be cast to client")
	cc.producer = &testProducer{publish: func(try bool, event publisher.Event) (queue.EntryID, bool) {
		return queue.EntryID(1), true
	}}

	c.PublishAll([]beat.Event{
		{Fields: mapstr.M{filterMeKey: true}, Meta: mapstr.M{}},
		{Fields: mapstr.M{filterMeKey: true}, Meta: mapstr.M{}},
		{Fields: mapstr.M{filterMeKey: true}, Meta: mapstr.M{}},
	})
	c.Publish(beat.Event{Meta: mapstr.M{}})
	require.NoError(t, c.Close())

	if clientCfg.ClientListener != nil {
		got, ok := clientCfg.ClientListener.(*mockClientListener)
		require.Truef(t, ok, "ClientListener must be of type %T, but got %T",
			&mockClientListener{},
			clientCfg.ClientListener)
		assert.Equal(t,
			got.eventsTotal,
			got.eventsFiltered+got.eventsPublished,
			"total events should be them sum of filtered, dropped and published"+
				"events")
		assert.Equal(t, 3, got.eventsFiltered,
			"should have 3 filtered events")
		assert.Equal(t, 1, got.eventsPublished,
			"should have 1 published event")
	}
}

type testProcessor struct {
	name        string
	processorFn func(in *beat.Event) (event *beat.Event, err error)
}

func (p *testProcessor) String() string {
	return "testProcessor-" + p.name
}

func (p *testProcessor) Run(in *beat.Event) (event *beat.Event, err error) {
	return p.processorFn(in)
}

type processorList struct {
	processors []beat.Processor
}

func (p processorList) String() string {
	var names []string
	for _, processor := range p.processors {
		names = append(names, processor.String())
	}

	return strings.Join(names, ", ")
}

func (p processorList) Run(in *beat.Event) (event *beat.Event, err error) {
	for _, processor := range p.processors {
		in, err = processor.Run(in)
		if in == nil {
			return nil, err
		}
	}

	return in, nil
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
	return []string{p.String()}
}

// Close the processor supporter
func (p testProcessorSupporter) Close() error {
	return processors.Close(p.Processor)
}

type mockClientListener struct {
	eventsTotal            int
	eventsFiltered         int
	eventsPublished        int
	eventsDroppedOnPublish int
}

func (m *mockClientListener) Closing() {}
func (m *mockClientListener) Closed()  {}
func (m *mockClientListener) NewEvent() {
	m.eventsTotal++
}
func (m *mockClientListener) Filtered() {
	m.eventsFiltered++
}
func (m *mockClientListener) Published() {
	m.eventsPublished++
}
func (m *mockClientListener) DroppedOnPublish(beat.Event) {
	m.eventsDroppedOnPublish++
}
