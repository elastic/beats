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

// This file was contributed to by generative AI

//nolint:errcheck // It's a test file
package input_logfile

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/input/filestream/internal/task"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/libbeat/tests/resources"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestReaderGroup(t *testing.T) {
	requireGroupSuccess := func(t *testing.T, ctx context.Context, cf context.CancelFunc, err error) {
		require.NotNil(t, ctx)
		require.NotNil(t, cf)
		require.Nil(t, err)
	}

	requireGroupError := func(t *testing.T, ctx context.Context, cf context.CancelFunc, err error) {
		require.Nil(t, ctx)
		require.Nil(t, cf)
		require.Error(t, err)
	}

	t.Run("assert new group is empty", func(t *testing.T) {
		rg := newReaderGroup()
		require.Equal(t, 0, len(rg.table))
	})

	t.Run("assert non existent key can be removed", func(t *testing.T) {
		rg := newReaderGroup()
		require.Equal(t, 0, len(rg.table))
		rg.remove("no such id")
		require.Equal(t, 0, len(rg.table))
	})

	t.Run("assert inserting existing key returns error", func(t *testing.T) {
		rg := newReaderGroup()
		ctx, cf, err := rg.newContext("test-id", context.Background())
		requireGroupSuccess(t, ctx, cf, err)
		require.Equal(t, 1, len(rg.table))

		newCtx, newCf, err := rg.newContext("test-id", context.Background())
		requireGroupError(t, newCtx, newCf, err)
	})

	t.Run("assert new key is added, can be removed and its context is cancelled", func(t *testing.T) {
		rg := newReaderGroup()
		ctx, cf, err := rg.newContext("test-id", context.Background())
		requireGroupSuccess(t, ctx, cf, err)
		require.Equal(t, 1, len(rg.table))

		require.Nil(t, ctx.Err())
		rg.remove("test-id")

		require.Equal(t, 0, len(rg.table))
		require.Error(t, ctx.Err(), context.Canceled)

		newCtx, newCf, err := rg.newContext("test-id", context.Background())
		requireGroupSuccess(t, newCtx, newCf, err)
		require.Equal(t, 1, len(rg.table))
		require.Nil(t, newCtx.Err())
	})
}

func TestDefaultHarvesterGroup(t *testing.T) {
	source := &testSource{name: "/path/to/test"}

	requireSourceAddedToBookkeeper := func(t *testing.T, hg *defaultHarvesterGroup, s Source) {
		require.True(t, hg.readers.hasID(hg.identifier.ID(s)))
	}

	requireSourceRemovedFromBookkeeper := func(t *testing.T, hg *defaultHarvesterGroup, s Source) {
		require.False(t, hg.readers.hasID(hg.identifier.ID(s)))
	}

	t.Run("assert a harvester is started in a goroutine", func(t *testing.T) {
		var wg sync.WaitGroup

		mockHarvester := &mockHarvester{onRun: correctOnRun, wg: &wg}
		hg := testDefaultHarvesterGroup(t, mockHarvester)

		goroutinesChecker := resources.NewGoroutinesChecker()
		defer goroutinesChecker.WaitUntilOriginalCount()

		wg.Add(1)
		ctx := input.Context{Logger: logp.L(), Cancelation: t.Context()}.WithStatusReporter(mockStatusReporter{})
		hg.Start(ctx, source)

		// wait until harvester.Run is done
		wg.Wait()
		// wait until goroutine that started `harvester.Run` is finished
		goroutinesChecker.WaitUntilOriginalCount()

		require.Equal(t, 1, mockHarvester.getRunCount())

		requireSourceRemovedFromBookkeeper(t, hg, source)
		// stopped source can be stopped
		require.Nil(t, hg.StopHarvesters())
	})

	t.Run("assert a harvester is only started if harvester limit haven't been reached", func(t *testing.T) {
		var wg sync.WaitGroup
		var harvesterRunningCount atomic.Int64
		var harvester1Finished, harvester2Finished atomic.Bool
		done1, done2 := make(chan struct{}), make(chan struct{})

		harvesterRun := func(_ input.Context, _ Source, _ Cursor, _ Publisher) error {
			harvesterRunningCount.Add(1)
			defer harvesterRunningCount.Add(-1)

			// it's the 2nd harvester, wait only on done2
			if harvester1Finished.Load() {
				<-done2
				harvester2Finished.Store(true)
				return nil
			}

			// it's the 1st harvester, wait until released
			<-done1
			harvester1Finished.Store(true)

			return nil
		}

		mockHarvester := &mockHarvester{
			onRun: harvesterRun,
			wg:    &wg,
		}
		hg := testDefaultHarvesterGroup(t, mockHarvester)
		hg.tg = task.NewGroup(1, time.Second, &logp.Logger{}, "")

		goroutinesChecker := resources.NewGoroutinesChecker()
		defer goroutinesChecker.WaitUntilOriginalCount()

		source1 := &testSource{name: "/path/to/test/1"}
		source2 := &testSource{name: "/path/to/test/2"}
		wg.Add(2)
		ctx := input.Context{Logger: logp.L(), Cancelation: t.Context()}.WithStatusReporter(mockStatusReporter{})
		hg.Start(ctx, source1)
		hg.Start(ctx, source2)

		assert.Eventually(t,
			func() bool {
				return harvesterRunningCount.Load() == 1 && harvesterRunningCount.Load() < 2
			},
			500*time.Minute,
			time.Millisecond)

		// release 1st harvester and wait for the 2nd to start
		close(done1)
		assert.Eventually(t,
			func() bool { return harvester1Finished.Load() },
			500*time.Minute,
			time.Millisecond)

		// wait harvester 2 to start
		assert.Eventually(t,
			func() bool {
				return harvesterRunningCount.Load() == 1 && harvesterRunningCount.Load() < 2
			},
			500*time.Minute,
			time.Millisecond)

		close(done2) // release harvester 2 to finish
		assert.Eventually(t,
			func() bool { return harvester2Finished.Load() },
			500*time.Minute,
			time.Millisecond)

		// wait until all harvester.Run are done
		wg.Wait()
		// wait until goroutine that started `harvester.Run` is finished
		goroutinesChecker.WaitUntilOriginalCount()

		require.Equal(t, 2, mockHarvester.getRunCount())

		requireSourceRemovedFromBookkeeper(t, hg, source1)
		requireSourceRemovedFromBookkeeper(t, hg, source2)

		// stopped source can be stopped
		require.Nil(t, hg.StopHarvesters())
	})

	t.Run("assert a harvester can be stopped and removed from bookkeeper", func(t *testing.T) {
		mockHarvester := &mockHarvester{onRun: blockUntilCancelOnRun}
		hg := testDefaultHarvesterGroup(t, mockHarvester)

		goroutinesChecker := resources.NewGoroutinesChecker()

		ctx := input.Context{Logger: logp.L(), Cancelation: t.Context()}.WithStatusReporter(mockStatusReporter{})
		hg.Start(ctx, source)

		goroutinesChecker.WaitUntilIncreased(1)
		// wait until harvester is started
		require.Eventually(t,
			func() bool { return mockHarvester.getRunCount() == 1 },
			5*time.Second,
			10*time.Millisecond,
			"run count must equal one")
		requireSourceAddedToBookkeeper(t, hg, source)
		// after started, stop it
		hg.Stop(source)
		_, err := goroutinesChecker.WaitUntilOriginalCount()
		require.NoError(t, err)
		requireSourceRemovedFromBookkeeper(t, hg, source)
	})

	t.Run("assert a harvester for same source cannot be started", func(t *testing.T) {
		mockHarvester := &mockHarvester{onRun: blockUntilCancelOnRun}
		hg := testDefaultHarvesterGroup(t, mockHarvester)
		inputCtx := input.Context{Logger: logp.L(), Cancelation: t.Context()}.WithStatusReporter(mockStatusReporter{})

		goroutinesChecker := resources.NewGoroutinesChecker()
		defer goroutinesChecker.WaitUntilOriginalCount()

		hg.Start(inputCtx, source)
		hg.Start(inputCtx, source)

		goroutinesChecker.WaitUntilIncreased(2)
		// error is expected as a harvester group was expected to start twice for the same source
		for !hg.readers.hasID(hg.identifier.ID(source)) {
		}
		time.Sleep(3 * time.Millisecond)

		hg.Stop(source)

		err := hg.StopHarvesters()
		require.NoError(t, err)

		require.Equal(t, 1, mockHarvester.getRunCount())
	})

	t.Run("assert a harvester panic is handled", func(t *testing.T) {
		mockHarvester := &mockHarvester{onRun: panicOnRun}
		hg := testDefaultHarvesterGroup(t, mockHarvester)
		defer func() {
			if v := recover(); v != nil {
				t.Errorf("did not recover from harvester panic in defaultHarvesterGroup")
			}
		}()

		goroutinesChecker := resources.NewGoroutinesChecker()

		ctx := input.Context{Logger: logp.L(), Cancelation: t.Context()}.WithStatusReporter(mockStatusReporter{})
		hg.Start(ctx, source)

		// wait until harvester is stopped
		goroutinesChecker.WaitUntilOriginalCount()

		// make sure harvester had run once
		require.Equal(t, 1, mockHarvester.getRunCount())
		requireSourceRemovedFromBookkeeper(t, hg, source)

		require.Nil(t, hg.StopHarvesters())
	})

	t.Run("assert a harvester error is handled", func(t *testing.T) {
		testLog := &testLogger{}
		mockHarvester := &mockHarvester{onRun: errorOnRun}
		hg := testDefaultHarvesterGroup(t, mockHarvester)
		hg.tg = task.NewGroup(0, 100*time.Millisecond, testLog, "")

		goroutinesChecker := resources.NewGoroutinesChecker()
		defer goroutinesChecker.WaitUntilOriginalCount()

		ctx := input.Context{Logger: logp.L(), Cancelation: t.Context()}.WithStatusReporter(mockStatusReporter{})
		hg.Start(ctx, source)

		goroutinesChecker.WaitUntilOriginalCount()

		requireSourceRemovedFromBookkeeper(t, hg, source)

		err := hg.StopHarvesters()
		assert.NoError(t, err)

		assert.Contains(t, testLog.String(), errHarvester.Error())
	})

	t.Run("assert already locked resource has to wait", func(t *testing.T) {
		var wg sync.WaitGroup
		mockHarvester := &mockHarvester{onRun: correctOnRun, wg: &wg}
		hg := testDefaultHarvesterGroup(t, mockHarvester)
		inputCtx := input.Context{Logger: logp.L(), Cancelation: t.Context()}.WithStatusReporter(mockStatusReporter{})

		r, err := lock(inputCtx, hg.store, hg.identifier.ID(source))
		if err != nil {
			t.Fatalf("cannot lock source")
		}

		goroutinesChecker := resources.NewGoroutinesChecker()

		wg.Add(1)
		hg.Start(inputCtx, source)

		goroutinesChecker.WaitUntilIncreased(1)
		ok := false
		for !ok {
			// wait until harvester is added to the bookeeper
			ok = hg.readers.hasID(hg.identifier.ID(source))
			if ok {
				releaseResource(r)
			}
		}

		// wait until harvester.Run is done
		wg.Wait()
		// wait until goroutine that started `harvester.Run` is finished
		goroutinesChecker.WaitUntilOriginalCount()
		require.Equal(t, 1, mockHarvester.getRunCount())
		require.Nil(t, hg.StopHarvesters())
	})

	t.Run("assert already locked resource has no problem when harvestergroup is cancelled", func(t *testing.T) {
		testLog := &testLogger{}
		mockHarvester := &mockHarvester{onRun: correctOnRun}
		hg := testDefaultHarvesterGroup(t, mockHarvester)
		hg.tg = task.NewGroup(0, 50*time.Millisecond, testLog, "")
		inputCtx := input.Context{Logger: logp.L(), Cancelation: t.Context()}.WithStatusReporter(mockStatusReporter{})

		goroutinesChecker := resources.NewGoroutinesChecker()
		defer goroutinesChecker.WaitUntilOriginalCount()

		r, err := lock(inputCtx, hg.store, hg.identifier.ID(source))
		if err != nil {
			t.Fatalf("cannot lock source")
		}
		defer releaseResource(r)

		hg.Start(inputCtx, source)

		goroutinesChecker.WaitUntilIncreased(1)
		assert.NoError(t, hg.StopHarvesters())

		assert.Equal(t, 0, mockHarvester.getRunCount())
	})

	t.Run("assert harvester can be restarted", func(t *testing.T) {
		var wg sync.WaitGroup
		mockHarvester := &mockHarvester{onRun: blockUntilCancelOnRun, wg: &wg}
		hg := testDefaultHarvesterGroup(t, mockHarvester)
		inputCtx := input.Context{Logger: logp.L(), Cancelation: t.Context()}.WithStatusReporter(mockStatusReporter{})

		goroutinesChecker := resources.NewGoroutinesChecker()
		defer goroutinesChecker.WaitUntilOriginalCount()

		wg.Add(2)
		hg.Start(inputCtx, source)
		hasRun := mockHarvester.getRunCount()
		for hasRun == 0 {
			hasRun = mockHarvester.getRunCount()
		}
		hg.Restart(inputCtx, source)

		for hasRun != 2 {
			hasRun = mockHarvester.getRunCount()
		}
		require.NoError(t, hg.StopHarvesters())

		wg.Wait()

		require.Equal(t, 2, mockHarvester.getRunCount())
	})
}

func TestCursorAllEventsPublished(t *testing.T) {
	fieldKey := "foo bar"
	var wg sync.WaitGroup
	source := &testSource{name: "/path/to/fake/file"}

	cursorCh := make(chan Cursor)
	publishLock := make(chan struct{})
	donePublishing := make(chan struct{})
	runFn := func(ctx input.Context, s Source, c Cursor, p Publisher) error {
		// Once the harvester is started, we send the cursor on the channel
		// so the test has access to it and can it proceed
		cursorCh <- c
		<-publishLock
		p.Publish(
			beat.Event{
				Timestamp: time.Now(),
				Fields: mapstr.M{
					// Add a known field so we can identify this event later on
					fieldKey: t.Name(),
				},
			}, c)
		donePublishing <- struct{}{}
		return nil
	}

	var cursor Cursor
	mockHarvester := &mockHarvester{onRun: runFn, wg: &wg}
	hg := testDefaultHarvesterGroup(t, mockHarvester)
	hg.pipeline = &MockPipeline{
		// Define the callback that will be called before each event is
		// published/acknowledged, when this callback is called, the
		// resource is still 'pending' on this acknowledgement.
		// So resource.pending must be 2, the input 'lock' and this pending
		// acknowledgement.
		//
		// This callback runs on a different goroutine, therefore we cannot
		// call t.FailNow and friends.
		publishCallback: func(e beat.Event) {
			// Ensure we have the correct event
			if ok, _ := e.Fields.HasKey(fieldKey); ok {
				uop, ok := e.Private.(*updateOp)
				if !ok {
					return
				}
				evtResource := uop.resource.key
				cursorKey := cursor.resource.key

				// Just to be on the safe side, ensure the event belongs to
				// the resource we're testing.
				if evtResource != cursorKey {
					t.Errorf(
						"cursor key %q and event resource key %q must be the same.",
						cursorKey, logp.EventType)
				}
				// cursor.resource.pending must be 2 here and
				// cursor.AllEventsPublished must return false
				if cursor.AllEventsPublished() {
					t.Errorf(
						"not all events have been published, pending events: %d",
						cursor.resource.pending.Load(),
					)
				}
			}
		}}

	wg.Add(1)
	testLogger := logptest.NewFileLogger(
		t,
		filepath.Join("..", "..", "..", "..", "build", "integration-tests"),
	)
	hg.Start(
		input.Context{
			Logger:      testLogger.Logger,
			Cancelation: t.Context(),
		},
		source)

	// Wait for the harvester to start and send us its resource
	cursor = <-cursorCh

	// As soon as the harvester starts, 'pending' must be 1
	// because the harvester locked the resource and no events
	// have been published yet.
	require.True(
		t,
		cursor.AllEventsPublished(),
		"All events must be published")

	// Ensure the harvester has the resource locked
	require.EqualValues(
		t,
		1,
		cursor.resource.pending.Load(),
		"While the harvester is running the resource must be locked, 'pending' must be 1")

	// Let the harvester call publish
	publishLock <- struct{}{}

	// Wait for the harvester to finish publishing
	<-donePublishing

	// Then wait for harvester.Run to return.
	// wg.Done is called by mockHarvester.Run, but the resurce
	// is released after mockHarvester.Run returns
	wg.Wait()

	// Once the harvester is closed, cursor.AllEventsPublished() must still
	// return true
	require.True(
		t,
		cursor.AllEventsPublished(),
		"cursor.AllEventsPublished() must return true when the harvester is closed.")

	// Ensure the resource has been released.
	// We know this log line is logged AFTER the resource is released
	testLogger.WaitLogsContains(
		t,
		"Stopped harvester for file",
		time.Second,
		"harvester did not stop")

	// Ensure the harvester has released the resource
	require.EqualValues(
		t,
		0,
		cursor.resource.pending.Load(),
		"once the harvester is done, the resource must be unlocked, 'pending' must be 0")
}

func testDefaultHarvesterGroup(t *testing.T, mockHarvester Harvester) *defaultHarvesterGroup {
	return &defaultHarvesterGroup{
		readers:    newReaderGroup(),
		pipeline:   &MockPipeline{},
		harvester:  mockHarvester,
		store:      testOpenStore(t, "test", nil),
		identifier: &SourceIdentifier{"filestream::.global::"},
		tg:         task.NewGroup(0, time.Second, logp.L(), ""),
	}
}

type mockHarvester struct {
	mu       sync.Mutex
	runCount int

	wg    *sync.WaitGroup
	onRun func(input.Context, Source, Cursor, Publisher) error
}

func (m *mockHarvester) Run(ctx input.Context, s Source, c Cursor, p Publisher, metrics *Metrics) error {
	if m.wg != nil {
		defer m.wg.Done()
	}

	m.mu.Lock()
	m.runCount += 1
	m.mu.Unlock()

	if m.onRun != nil {
		return m.onRun(ctx, s, c, p)
	}
	return nil
}

func (m *mockHarvester) getRunCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.runCount
}

func (m *mockHarvester) Test(_ Source, _ input.TestContext) error { return nil }

func (m *mockHarvester) Name() string { return "mock" }

func correctOnRun(_ input.Context, _ Source, _ Cursor, _ Publisher) error {
	return nil
}

func blockUntilCancelOnRun(c input.Context, _ Source, _ Cursor, _ Publisher) error {
	<-c.Cancelation.Done()
	return nil
}

var errHarvester = fmt.Errorf("harvester error")

func errorOnRun(_ input.Context, _ Source, _ Cursor, _ Publisher) error {
	return errHarvester
}

func panicOnRun(_ input.Context, _ Source, _ Cursor, _ Publisher) error {
	panic("don't panic")
}

type testLogger strings.Builder

func (tl *testLogger) Errorf(format string, args ...any) {
	sb := (*strings.Builder)(tl)
	fmt.Fprintf(sb, format, args...)
	fmt.Fprint(sb, "\n")
}

func (tl *testLogger) String() string {
	return (*strings.Builder)(tl).String()
}

// MockClient is a mock implementation of the beat.Client interface.
type MockClient struct {
	published []beat.Event // Slice to store published events

	closed          bool               // Flag to indicate if the client is closed
	mu              sync.Mutex         // Mutex to synchronize access to the published events slice
	publishCallback func(e beat.Event) // Callback called when the client is publishing the event, but before acknowledging it
}

// GetEvents returns all the events published by the mock client.
func (m *MockClient) GetEvents() []beat.Event {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.published
}

// Publish publishes a single event.
func (m *MockClient) Publish(e beat.Event) {
	es := make([]beat.Event, 1)
	es = append(es, e)

	m.PublishAll(es)
}

// PublishAll publishes multiple events.
func (m *MockClient) PublishAll(es []beat.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, evt := range es {
		if m.publishCallback != nil {
			m.publishCallback(evt)
		}

		// If there is an update operation on this event, acknowledge it.
		if op, ok := evt.Private.(*updateOp); ok {
			op.done(1)
		}
	}

	m.published = append(m.published, es...)
}

// Close closes the mock client.
func (m *MockClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("mock already closed")
	}

	m.closed = true
	return nil
}

// MockPipeline is a mock implementation of the beat.Pipeline interface.
type MockPipeline struct {
	c               beat.Client        // Client used by the pipeline
	mu              sync.Mutex         // Mutex to synchronize access to the client
	publishCallback func(e beat.Event) // Callback called when the client is publishing the event, but before acknowledging it
}

// ConnectWith connects the mock pipeline with a client using the provided configuration.
func (mp *MockPipeline) ConnectWith(config beat.ClientConfig) (beat.Client, error) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	c := &MockClient{}
	if mp.publishCallback != nil {
		c.publishCallback = mp.publishCallback
	}

	mp.c = c

	return c, nil
}

// Connect connects the mock pipeline with a client using the default configuration.
func (mp *MockPipeline) Connect() (beat.Client, error) {
	return mp.ConnectWith(beat.ClientConfig{})
}

type mockStatusReporter struct{}

// UpdateStatus is a no-op
func (m mockStatusReporter) UpdateStatus(status status.Status, msg string) {
}
