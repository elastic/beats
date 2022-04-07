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

//nolint: errcheck // It's a test file
package input_logfile

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	input "github.com/elastic/beats/v8/filebeat/input/v2"
	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/libbeat/tests/resources"
	"github.com/elastic/beats/v8/x-pack/dockerlogbeat/pipelinemock"
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

	t.Run("assert new harvester cannot be added if limit is reached", func(t *testing.T) {
		rg := newReaderGroupWithLimit(1)
		require.Equal(t, 0, len(rg.table))
		ctx, cf, err := rg.newContext("test-id", context.Background())
		requireGroupSuccess(t, ctx, cf, err)
		ctx, cf, err = rg.newContext("test-id", context.Background())
		requireGroupError(t, ctx, cf, err)
	})
}

func TestDefaultHarvesterGroup(t *testing.T) {
	t.Skip("flaky test: https://github.com/elastic/beats/issues/26727")
	source := &testSource{"/path/to/test"}

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

		gorountineChecker := resources.NewGoroutinesChecker()
		defer gorountineChecker.WaitUntilOriginalCount()

		wg.Add(1)
		hg.Start(input.Context{Logger: logp.L(), Cancelation: context.Background()}, source)

		// wait until harvester.Run is done
		wg.Wait()
		// wait until goroutine that started `harvester.Run` is finished
		gorountineChecker.WaitUntilOriginalCount()

		require.Equal(t, 1, mockHarvester.getRunCount())

		requireSourceRemovedFromBookkeeper(t, hg, source)
		// stopped source can be stopped
		require.Nil(t, hg.StopGroup())
	})

	t.Run("assert a harvester can be stopped and removed from bookkeeper", func(t *testing.T) {
		t.Skip("flaky test: https://github.com/elastic/beats/issues/25805")
		mockHarvester := &mockHarvester{onRun: blockUntilCancelOnRun}
		hg := testDefaultHarvesterGroup(t, mockHarvester)

		gorountineChecker := resources.NewGoroutinesChecker()

		hg.Start(input.Context{Logger: logp.L(), Cancelation: context.Background()}, source)

		gorountineChecker.WaitUntilIncreased(1)
		// wait until harvester is started
		if mockHarvester.getRunCount() == 1 {
			requireSourceAddedToBookkeeper(t, hg, source)
			// after started, stop it
			hg.Stop(source)
			gorountineChecker.WaitUntilOriginalCount()
		}

		requireSourceRemovedFromBookkeeper(t, hg, source)
	})

	t.Run("assert a harvester for same source cannot be started", func(t *testing.T) {
		mockHarvester := &mockHarvester{onRun: blockUntilCancelOnRun}
		hg := testDefaultHarvesterGroup(t, mockHarvester)
		inputCtx := input.Context{Logger: logp.L(), Cancelation: context.Background()}

		gorountineChecker := resources.NewGoroutinesChecker()
		defer gorountineChecker.WaitUntilOriginalCount()

		hg.Start(inputCtx, source)
		hg.Start(inputCtx, source)

		gorountineChecker.WaitUntilIncreased(2)
		// error is expected as a harvester group was expected to start twice for the same source
		for !hg.readers.hasID(hg.identifier.ID(source)) {
		}
		time.Sleep(3 * time.Millisecond)

		hg.Stop(source)

		err := hg.StopGroup()
		require.Error(t, err)

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

		gorountineChecker := resources.NewGoroutinesChecker()

		hg.Start(input.Context{Logger: logp.L(), Cancelation: context.Background()}, source)

		// wait until harvester is stopped
		gorountineChecker.WaitUntilOriginalCount()

		// make sure harvester had run once
		require.Equal(t, 1, mockHarvester.getRunCount())
		requireSourceRemovedFromBookkeeper(t, hg, source)

		require.Nil(t, hg.StopGroup())
	})

	t.Run("assert a harvester error is handled", func(t *testing.T) {
		mockHarvester := &mockHarvester{onRun: errorOnRun}
		hg := testDefaultHarvesterGroup(t, mockHarvester)

		gorountineChecker := resources.NewGoroutinesChecker()
		defer gorountineChecker.WaitUntilOriginalCount()

		hg.Start(input.Context{Logger: logp.L(), Cancelation: context.Background()}, source)

		gorountineChecker.WaitUntilOriginalCount()

		requireSourceRemovedFromBookkeeper(t, hg, source)

		err := hg.StopGroup()
		require.Error(t, err)
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

		gorountineChecker := resources.NewGoroutinesChecker()

		wg.Add(1)
		hg.Start(inputCtx, source)

		gorountineChecker.WaitUntilIncreased(1)
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
		gorountineChecker.WaitUntilOriginalCount()
		require.Equal(t, 1, mockHarvester.getRunCount())
		require.Nil(t, hg.StopGroup())
	})

	t.Run("assert already locked resource has no problem when harvestergroup is cancelled", func(t *testing.T) {
		mockHarvester := &mockHarvester{onRun: correctOnRun}
		hg := testDefaultHarvesterGroup(t, mockHarvester)
		inputCtx := input.Context{Logger: logp.L(), Cancelation: context.Background()}

		gorountineChecker := resources.NewGoroutinesChecker()
		defer gorountineChecker.WaitUntilOriginalCount()

		r, err := lock(inputCtx, hg.store, hg.identifier.ID(source))
		if err != nil {
			t.Fatalf("cannot lock source")
		}
		defer releaseResource(r)

		hg.Start(inputCtx, source)

		gorountineChecker.WaitUntilIncreased(1)
		require.Error(t, hg.StopGroup())

		require.Equal(t, 0, mockHarvester.getRunCount())
	})

	t.Run("assert harvester can be restarted", func(t *testing.T) {
		var wg sync.WaitGroup
		mockHarvester := &mockHarvester{onRun: blockUntilCancelOnRun, wg: &wg}
		hg := testDefaultHarvesterGroup(t, mockHarvester)
		inputCtx := input.Context{Logger: logp.L(), Cancelation: context.Background()}

		gorountineChecker := resources.NewGoroutinesChecker()
		defer gorountineChecker.WaitUntilOriginalCount()

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
		require.NoError(t, hg.StopGroup())

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
		tg:         unison.TaskGroup{},
	}
}

type mockHarvester struct {
	mu       sync.Mutex
	runCount int

	wg    *sync.WaitGroup
	onRun func(input.Context, Source, Cursor, Publisher) error
}

func (m *mockHarvester) Run(ctx input.Context, s Source, c Cursor, p Publisher) error {
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

func errorOnRun(_ input.Context, _ Source, _ Cursor, _ Publisher) error {
	return fmt.Errorf("harvester error")
}

func panicOnRun(_ input.Context, _ Source, _ Cursor, _ Publisher) error {
	panic("don't panic")
}
