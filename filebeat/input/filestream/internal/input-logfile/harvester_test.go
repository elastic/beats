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

package input_logfile

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/dockerlogbeat/pipelinemock"
	"github.com/elastic/go-concert/unison"
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
	source := &testSource{"/path/to/test"}

	t.Run("assert a harvester is started in a goroutine", func(t *testing.T) {
		var wg sync.WaitGroup
		mockHarvester := &mockHarvester{onRun: correctOnRun, wg: &wg}
		hg := testDefaultHarvesterGroup(t, mockHarvester)

		gorountineChecker := newGoroutineChecker()

		wg.Add(1)
		hg.Start(input.Context{Logger: logp.L(), Cancelation: context.Background()}, source)

		// wait until harvester.Run is done
		wg.Wait()
		// wait until goroutine that started `harvester.Run` is finished
		gorountineChecker.waitOriginalCount()

		require.Equal(t, 1, mockHarvester.runCount)

		// when finished removed from bookeeper
		_, ok := hg.readers.table[source.Name()]
		require.False(t, ok)
		// stopped source can be stopped
		require.Nil(t, hg.StopGroup())

		gorountineChecker.waitOriginalCount()
	})

	t.Run("assert a harvester can be stopped and removed from bookkeeper", func(t *testing.T) {
		mockHarvester := &mockHarvester{onRun: blockUntilCancelOnRun}
		hg := testDefaultHarvesterGroup(t, mockHarvester)

		gorountineChecker := newGoroutineChecker()

		hg.Start(input.Context{Logger: logp.L(), Cancelation: context.Background()}, source)

		// run commands while harvester is running
		gorountineChecker.waitWhileMoreGoroutinesWithFunc(func() {
			// wait until harvester is started
			if mockHarvester.runCount == 1 {
				// assert that it is part of the bookkeeper before it is stopped
				_, ok := hg.readers.table[source.Name()]
				require.True(t, ok)
				// after started, stop it
				hg.Stop(source)
				// wait until the harvester is stopped
				gorountineChecker.waitOriginalCount()
			}
		})

		// when finished removed from bookeeper
		_, ok := hg.readers.table[source.Name()]
		require.False(t, ok)
	})

	t.Run("assert a harvester for same source cannot be started", func(t *testing.T) {
		mockHarvester := &mockHarvester{onRun: blockUntilCancelOnRun}
		hg := testDefaultHarvesterGroup(t, mockHarvester)
		inputCtx := input.Context{Logger: logp.L(), Cancelation: context.Background()}
		gorountineChecker := newGoroutineChecker()

		hg.Start(inputCtx, source)
		hg.Start(inputCtx, source)

		gorountineChecker.waitWhileMoreGoroutinesWithFunc(func() {
			// error is expected as a harvester group was expected to start twice for the same source
			err := hg.StopGroup()
			require.Error(t, err)
		})

		require.Equal(t, 1, mockHarvester.runCount)
	})

	t.Run("assert a harvester panic is handled", func(t *testing.T) {
		mockHarvester := &mockHarvester{onRun: panicOnRun}
		hg := testDefaultHarvesterGroup(t, mockHarvester)
		defer func() {
			if v := recover(); v != nil {
				t.Errorf("did not recover from harvester panic in defaultHarvesterGroup")
			}
		}()

		gorountineChecker := newGoroutineChecker()

		hg.Start(input.Context{Logger: logp.L(), Cancelation: context.Background()}, source)

		require.Nil(t, hg.StopGroup())
		gorountineChecker.waitOriginalCount()
	})

	t.Run("assert a harvester error is handled", func(t *testing.T) {
		mockHarvester := &mockHarvester{onRun: errorOnRun}
		hg := testDefaultHarvesterGroup(t, mockHarvester)
		gorountineChecker := newGoroutineChecker()

		hg.Start(input.Context{Logger: logp.L(), Cancelation: context.Background()}, source)

		gorountineChecker.waitOriginalCount()

		_, ok := hg.readers.table[source.Name()]
		require.False(t, ok)

		err := hg.StopGroup()
		require.Error(t, err)
	})

	t.Run("assert already locked resource has to wait", func(t *testing.T) {
		var wg sync.WaitGroup
		mockHarvester := &mockHarvester{onRun: correctOnRun, wg: &wg}
		hg := testDefaultHarvesterGroup(t, mockHarvester)
		inputCtx := input.Context{Logger: logp.L(), Cancelation: context.Background()}

		r, err := lock(inputCtx, hg.store, source.Name())
		if err != nil {
			t.Fatalf("cannot lock source")
		}

		gorountineChecker := newGoroutineChecker()

		wg.Add(1)
		hg.Start(inputCtx, source)

		locked := true
		gorountineChecker.waitWhileMoreGoroutinesWithFunc(func() {
			// harvester is waiting to start
			_, ok := hg.readers.table[source.Name()]
			if ok && locked {
				releaseResource(r)
				locked = false
			}
		})

		// wait until harvester.Run is done
		wg.Wait()
		// wait until goroutine that started `harvester.Run` is finished
		gorountineChecker.waitOriginalCount()
		require.Equal(t, 1, mockHarvester.runCount)
		require.Nil(t, hg.StopGroup())
	})

	t.Run("assert already locked resource has no problem when harvestergroup is cancelled", func(t *testing.T) {
		mockHarvester := &mockHarvester{onRun: correctOnRun}
		hg := testDefaultHarvesterGroup(t, mockHarvester)
		inputCtx := input.Context{Logger: logp.L(), Cancelation: context.Background()}
		gorountineChecker := newGoroutineChecker()

		r, err := lock(inputCtx, hg.store, source.Name())
		if err != nil {
			t.Fatalf("cannot lock source")
		}
		defer releaseResource(r)

		hg.Start(inputCtx, source)

		gorountineChecker.waitWhileMoreGoroutinesWithFunc(func() {
			err := hg.StopGroup()
			require.Error(t, err)
		})

		require.Equal(t, 0, mockHarvester.runCount)

		gorountineChecker.waitOriginalCount()
	})
}

func testDefaultHarvesterGroup(t *testing.T, mockHarvester Harvester) *defaultHarvesterGroup {
	return &defaultHarvesterGroup{
		readers:   newReaderGroup(),
		pipeline:  &pipelinemock.MockPipelineConnector{},
		harvester: mockHarvester,
		store:     testOpenStore(t, "test", nil),
		tg:        unison.TaskGroup{},
	}
}

type gorountineChecker struct {
	n    int
	wait time.Duration
}

func newGoroutineChecker() *gorountineChecker {
	return &gorountineChecker{
		n:    runtime.NumGoroutine(),
		wait: 10 * time.Millisecond,
	}
}

func (c *gorountineChecker) waitWhileMoreGoroutinesWithFunc(f func()) {
	for c.n < runtime.NumGoroutine() {
		fmt.Println("waiting with function", c.n, runtime.NumGoroutine())
		time.Sleep(c.wait)

		f()
	}
}

func (c *gorountineChecker) waitOriginalCount() {
	fmt.Println("waiting until original", c.n, runtime.NumGoroutine())
	for c.n < runtime.NumGoroutine() {
		time.Sleep(c.wait)
	}
}

type mockHarvester struct {
	runCount int
	wg       *sync.WaitGroup
	onRun    func(input.Context, Source, Cursor, Publisher) error
}

func (m *mockHarvester) Run(ctx input.Context, s Source, c Cursor, p Publisher) error {
	if m.wg != nil {
		defer m.wg.Done()
	}

	m.runCount += 1
	if m.onRun != nil {
		return m.onRun(ctx, s, c, p)
	}
	return nil
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

func errorOnRun(_ input.Context, _ Source, _ Cursor, _ Publisher) error {
	return fmt.Errorf("harvester error")
}

func panicOnRun(_ input.Context, _ Source, _ Cursor, _ Publisher) error {
	panic(fmt.Errorf("don't panic"))
}
