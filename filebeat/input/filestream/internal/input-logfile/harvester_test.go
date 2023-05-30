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

//nolint:errcheck // It's a test file
package input_logfile

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/input/filestream/internal/task"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/tests/resources"
	"github.com/elastic/beats/v7/x-pack/dockerlogbeat/pipelinemock"
	"github.com/elastic/elastic-agent-libs/logp"
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
		hg.Start(input.Context{Logger: logp.L(), Cancelation: context.Background()}, source)

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
		var harvesterRunningCount atomic.Int
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
		hg.Start(
			input.Context{Logger: logp.L(), Cancelation: context.Background()},
			source1)
		hg.Start(
			input.Context{Logger: logp.L(), Cancelation: context.Background()},
			source2)

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

		hg.Start(input.Context{Logger: logp.L(), Cancelation: context.Background()}, source)

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
		inputCtx := input.Context{Logger: logp.L(), Cancelation: context.Background()}

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

		hg.Start(input.Context{Logger: logp.L(), Cancelation: context.Background()}, source)

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

		hg.Start(input.Context{Logger: logp.L(), Cancelation: context.Background()}, source)

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
		inputCtx := input.Context{Logger: logp.L(), Cancelation: context.Background()}

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
		inputCtx := input.Context{Logger: logp.L(), Cancelation: context.Background()}

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
		inputCtx := input.Context{Logger: logp.L(), Cancelation: context.Background()}

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

func testDefaultHarvesterGroup(t *testing.T, mockHarvester Harvester) *defaultHarvesterGroup {
	return &defaultHarvesterGroup{
		readers:    newReaderGroup(),
		pipeline:   &pipelinemock.MockPipelineConnector{},
		harvester:  mockHarvester,
		store:      testOpenStore(t, "test", nil),
		identifier: &sourceIdentifier{"filestream::.global::"},
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

func (tl *testLogger) Errorf(format string, args ...interface{}) {
	sb := (*strings.Builder)(tl)
	sb.WriteString(fmt.Sprintf(format, args...))
	sb.WriteString("\n")
}

func (tl *testLogger) String() string {
	return (*strings.Builder)(tl).String()
}
