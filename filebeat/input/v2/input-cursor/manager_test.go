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

package cursor

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/features"
	pubtest "github.com/elastic/beats/v7/libbeat/publisher/testing"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/tests/resources"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/go-concert/unison"
)

type fakeTestInput struct {
	OnTest func(Source, input.TestContext) error
	OnRun  func(input.Context, Source, Cursor, Publisher) error
}

type stringSource string

func TestManager_Init(t *testing.T) {
	// Integration style tests for the InputManager and the state garbage collector

	t.Run("stopping the taskgroup kills internal go-routines", func(t *testing.T) {
		numRoutines := runtime.NumGoroutine()

		var grp unison.TaskGroup
		store := createSampleStore(t, nil)
		manager := &InputManager{
			Logger:              logptest.NewTestingLogger(t, "test"),
			StateStore:          store,
			Type:                "test",
			DefaultCleanTimeout: 10 * time.Millisecond,
		}

		err := manager.Init(&grp)
		require.NoError(t, err)

		time.Sleep(200 * time.Millisecond)
		_ = grp.Stop()

		// wait for all go-routines to be gone

		for numRoutines < runtime.NumGoroutine() {
			time.Sleep(1 * time.Millisecond)
		}
	})

	t.Run("collect old entries after startup", func(t *testing.T) {
		store := createSampleStore(t, map[string]state{
			"test::key": {
				TTL:     1 * time.Millisecond,
				Updated: time.Now().Add(-24 * time.Hour),
			},
		})
		store.GCPeriod = 10 * time.Millisecond

		var grp unison.TaskGroup
		//nolint:errcheck // We don't need the error from grp.Stop()
		defer grp.Stop()
		manager := &InputManager{
			Logger:              logptest.NewTestingLogger(t, "test"),
			StateStore:          store,
			Type:                "test",
			DefaultCleanTimeout: 10 * time.Millisecond,
		}

		err := manager.Init(&grp)
		require.NoError(t, err)

		for len(store.snapshot()) > 0 {
			time.Sleep(1 * time.Millisecond)
		}
	})
}

func TestManager_InitDefersStoreForES(t *testing.T) {
	// Verify that ES-backed inputs defer store creation from Init() to Create().
	enableESStateStore(t)

	data := map[string]state{
		"test::mykey": {Cursor: "value1"},
	}
	stateStore := createSampleStore(t, data)

	var grp unison.TaskGroup
	defer grp.Stop() //nolint:errcheck // We don't need the error from grp.Stop()

	manager := esManager(t, stateStore)

	// Init() should not create a store for ES-backed inputs.
	err := manager.Init(&grp)
	require.NoError(t, err)
	assert.Empty(t, manager.stores, "no store should be opened by Init() for ES-backed inputs")

	// Create() should create the store with the inputID.
	_, err = manager.Create(conf.MustNewConfigFrom(map[string]any{
		"id": "my-input-id",
	}))
	require.NoError(t, err)
	assert.Len(t, manager.stores, 1, "store should be created after Create()")

	snap := storeMemorySnapshot(managerStore(t, manager, "my-input-id"))
	assert.Contains(t, snap, "test::mykey")
	assert.Equal(t, "value1", snap["test::mykey"].Cursor)
}

// Elasticsearch inputs with different IDs must use separate cursor stores,
// even when they have the same input type.
func TestManager_ESStorePerInputID(t *testing.T) {
	enableESStateStore(t)

	stateStore := createESStore(t)

	var grp unison.TaskGroup
	defer grp.Stop() //nolint:errcheck // We don't need the error from grp.Stop()

	manager := esManager(t, stateStore)
	require.NoError(t, manager.Init(&grp))

	for _, id := range []string{"input-a", "input-b"} {
		_, err := manager.Create(conf.MustNewConfigFrom(map[string]any{"id": id}))
		require.NoError(t, err, "Create must succeed for input id %q", id)
	}
	require.Len(t, manager.stores, 2, "each input id must get its own store")

	// State written to one input's store must not appear in the other store.
	storeA := managerStore(t, manager, "input-a")
	storeA.UpdateTTL(storeA.Get("test::input-a::mykey"), time.Minute)

	assert.Contains(t, stateStore.snapshotFor(t, "input-a"), "test::input-a::mykey",
		"input-a must persist its own cursor")
	assert.Empty(t, stateStore.snapshotFor(t, "input-b"),
		"input-b must not see the cursor written by input-a")
}

// enableESStateStore routes the "test" input type to the Elasticsearch state store.
func enableESStateStore(t *testing.T) {
	t.Helper()

	t.Setenv("AGENTLESS_ELASTICSEARCH_STATE_STORE_INPUT_TYPES", "test")
	features.ReinitForTest()
	t.Cleanup(func() { features.ReinitForTest() }) // restore after test
}

// esManager builds an InputManager for the ES-backed "test" input type.
func esManager(t *testing.T, stateStore statestore.States) *InputManager {
	t.Helper()

	return &InputManager{
		Logger:              logptest.NewTestingLogger(t, "test"),
		StateStore:          stateStore,
		Type:                "test",
		DefaultCleanTimeout: 30 * time.Minute,
		Configure: func(cfg *conf.C, log *logp.Logger) ([]Source, Input, error) {
			return sourceList("mykey"), &fakeTestInput{}, nil
		},
	}
}

// managerStore returns the store the manager opened for inputID.
func managerStore(t *testing.T, manager *InputManager, inputID string) *store {
	t.Helper()

	manager.mu.Lock()
	e, ok := manager.stores[manager.storeKey(inputID)]
	manager.mu.Unlock()
	require.True(t, ok, "no store for input id %q", inputID)

	<-e.ready
	require.NoError(t, e.err, "the store for input id %q must open", inputID)
	return e.store
}

// Opening a store can be slow: the Elasticsearch store waits for the output
// configuration before it reads any state. That must not delay Create for other
// input IDs, and concurrent Create calls for one ID must share one store.
func TestManager_ESStoreOpenDoesNotBlockOtherInputIDs(t *testing.T) {
	enableESStateStore(t)

	slowGate := make(chan struct{})
	releaseSlow := sync.OnceFunc(func() { close(slowGate) })
	stateStore := &gatedESStateStore{
		esStateStore: createESStore(t),
		gates:        map[string]chan struct{}{"slow": slowGate},
	}

	var grp unison.TaskGroup
	defer grp.Stop() //nolint:errcheck // We don't need the error from grp.Stop()

	manager := esManager(t, stateStore)
	require.NoError(t, manager.Init(&grp))

	var wg sync.WaitGroup
	// Unblock and wait for the slow Create calls before the registry closes.
	t.Cleanup(func() {
		releaseSlow()
		wg.Wait()
	})

	slowInputs := make([]input.Input, 4)
	for i := range slowInputs {
		wg.Go(func() {
			inp, err := manager.Create(conf.MustNewConfigFrom(map[string]any{"id": "slow"}))
			assert.NoError(t, err, "Create for input id %q must succeed", "slow")
			slowInputs[i] = inp
		})
	}
	require.Eventually(t, func() bool { return stateStore.storeForCalls("slow") > 0 },
		10*time.Second, time.Millisecond, "the store for input id %q must start opening", "slow")

	fastDone := make(chan error, 1)
	go func() {
		_, err := manager.Create(conf.MustNewConfigFrom(map[string]any{"id": "fast"}))
		fastDone <- err
	}()
	select {
	case err := <-fastDone:
		assert.NoError(t, err, "Create for input id %q must succeed", "fast")
	case <-time.After(10 * time.Second):
		t.Fatalf("Create for input id %q waited for the store of input id %q to open", "fast", "slow")
	}

	releaseSlow()
	wg.Wait()

	assert.Equal(t, 1, stateStore.storeForCalls("slow"),
		"concurrent Create calls for one input id must open its store once")
	first, ok := slowInputs[0].(*managedInput)
	require.True(t, ok, "Create for input id %q must return a managed input", "slow")
	for i, inp := range slowInputs {
		managed, ok := inp.(*managedInput)
		if assert.True(t, ok, "Create call %d for input id %q must return a managed input", i, "slow") {
			assert.Same(t, first.store, managed.store,
				"Create call %d for input id %q must share the store of the first call", i, "slow")
		}
	}
}

func TestManager_Create(t *testing.T) {
	t.Run("fail if no source is configured", func(t *testing.T) {
		manager := constInput(t, nil, &fakeTestInput{})
		_, err := manager.Create(conf.NewConfig())
		require.Error(t, err)
	})

	t.Run("fail if config error", func(t *testing.T) {
		manager := failingManager(t, errors.New("oops"))
		_, err := manager.Create(conf.NewConfig())
		require.Error(t, err)
	})

	t.Run("fail if no input runner is returned", func(t *testing.T) {
		manager := constInput(t, sourceList("test"), nil)
		_, err := manager.Create(conf.NewConfig())
		require.Error(t, err)
	})

	t.Run("configure ok", func(t *testing.T) {
		manager := constInput(t, sourceList("test"), &fakeTestInput{})
		_, err := manager.Create(conf.NewConfig())
		require.NoError(t, err)
	})

	t.Run("configuring inputs with overlapping sources is allowed", func(t *testing.T) {
		manager := simpleManagerWithConfigure(t, func(cfg *conf.C, log *logp.Logger) ([]Source, Input, error) {
			config := struct{ Sources []string }{}
			err := cfg.Unpack(&config)
			return sourceList(config.Sources...), &fakeTestInput{}, err
		})

		_, err := manager.Create(conf.MustNewConfigFrom(map[string]any{
			"sources": []string{"a"},
		}))
		require.NoError(t, err)

		_, err = manager.Create(conf.MustNewConfigFrom(map[string]any{
			"sources": []string{"a"},
		}))
		require.NoError(t, err)
	})
}

func TestManager_InputsTest(t *testing.T) {
	var mu sync.Mutex
	var seen []string

	sources := sourceList("source1", "source2")

	t.Run("test is run for each source", func(t *testing.T) {
		defer resources.NewGoroutinesChecker().Check(t)

		manager := constInput(t, sources, &fakeTestInput{
			OnTest: func(source Source, _ input.TestContext) error {
				mu.Lock()
				defer mu.Unlock()
				seen = append(seen, source.Name())
				return nil
			},
		})

		inp, err := manager.Create(conf.NewConfig())
		require.NoError(t, err)

		err = inp.Test(input.TestContext{})
		require.NoError(t, err)

		sort.Strings(seen)
		require.Equal(t, []string{"source1", "source2"}, seen)
	})

	t.Run("cancel gets distributed to all source tests", func(t *testing.T) {
		defer resources.NewGoroutinesChecker().Check(t)

		manager := constInput(t, sources, &fakeTestInput{
			OnTest: func(_ Source, ctx input.TestContext) error {
				<-ctx.Cancelation.Done()
				return nil
			},
		})

		inp, err := manager.Create(conf.NewConfig())
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.TODO())

		var wg sync.WaitGroup
		wg.Go(func() {
			err = inp.Test(input.TestContext{Cancelation: ctx})
		})

		cancel()
		wg.Wait()
		require.NoError(t, err)
	})

	t.Run("fail if test for one source fails", func(t *testing.T) {
		defer resources.NewGoroutinesChecker().Check(t)

		failing := Source(stringSource("source1"))
		sources := []Source{failing, stringSource("source2")}

		manager := constInput(t, sources, &fakeTestInput{
			OnTest: func(source Source, _ input.TestContext) error {
				if source == failing {
					t.Log("return error")
					return errors.New("oops")
				}
				t.Log("return ok")
				return nil
			},
		})

		inp, err := manager.Create(conf.NewConfig())
		require.NoError(t, err)

		var wg sync.WaitGroup
		wg.Go(func() {
			err = inp.Test(input.TestContext{})
			t.Logf("Test returned: %v", err)
		})

		wg.Wait()
		require.Error(t, err)
	})

	t.Run("panic is captured", func(t *testing.T) {
		defer resources.NewGoroutinesChecker().Check(t)

		manager := constInput(t, sources, &fakeTestInput{
			OnTest: func(source Source, _ input.TestContext) error {
				panic("oops")
			},
		})

		inp, err := manager.Create(conf.NewConfig())
		require.NoError(t, err)

		var wg sync.WaitGroup
		wg.Go(func() {
			err = inp.Test(input.TestContext{Logger: logptest.NewTestingLogger(t, "test")})
			t.Logf("Test returned: %v", err)
		})

		wg.Wait()
		require.Error(t, err)
	})
}

func TestManager_InputsRun(t *testing.T) {
	// Integration style tests for the InputManager and Input.Run

	t.Run("input returned with error", func(t *testing.T) {
		defer resources.NewGoroutinesChecker().Check(t)

		manager := constInput(t, sourceList("test"), &fakeTestInput{
			OnRun: func(_ input.Context, _ Source, _ Cursor, _ Publisher) error {
				return errors.New("oops")
			},
		})

		inp, err := manager.Create(conf.NewConfig())
		require.NoError(t, err)

		cancelCtx := t.Context()

		var clientCounters pubtest.ClientCounter
		id := uuid.Must(uuid.NewV4()).String()
		ctx := input.Context{
			ID:              id,
			IDWithoutName:   id,
			Name:            inp.Name(),
			Cancelation:     cancelCtx,
			MetricsRegistry: monitoring.NewRegistry(),
			Logger:          manager.Logger,
		}
		err = inp.Run(ctx, clientCounters.BuildConnector())
		require.Error(t, err)
		require.Equal(t, 0, clientCounters.Active())
	})

	t.Run("panic is captured", func(t *testing.T) {
		defer resources.NewGoroutinesChecker().Check(t)

		manager := constInput(t, sourceList("test"), &fakeTestInput{
			OnRun: func(_ input.Context, _ Source, _ Cursor, _ Publisher) error {
				panic("oops")
			},
		})

		inp, err := manager.Create(conf.NewConfig())
		require.NoError(t, err)

		cancelCtx := t.Context()

		var clientCounters pubtest.ClientCounter
		id := uuid.Must(uuid.NewV4()).String()
		ctx := input.Context{
			ID:              id,
			IDWithoutName:   id,
			Name:            inp.Name(),
			Cancelation:     cancelCtx,
			MetricsRegistry: monitoring.NewRegistry(),
			Logger:          manager.Logger,
		}
		err = inp.Run(ctx, clientCounters.BuildConnector())
		require.Error(t, err)
		require.Equal(t, 0, clientCounters.Active())
	})

	t.Run("shutdown on signal", func(t *testing.T) {
		defer resources.NewGoroutinesChecker().Check(t)

		manager := constInput(t, sourceList("test"), &fakeTestInput{
			OnRun: func(ctx input.Context, _ Source, _ Cursor, _ Publisher) error {
				<-ctx.Cancelation.Done()
				return nil
			},
		})

		inp, err := manager.Create(conf.NewConfig())
		require.NoError(t, err)

		cancelCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var clientCounters pubtest.ClientCounter
		var wg sync.WaitGroup
		wg.Go(func() {
			id := uuid.Must(uuid.NewV4()).String()
			ctx := input.Context{
				ID:              id,
				IDWithoutName:   id,
				Name:            inp.Name(),
				Cancelation:     cancelCtx,
				MetricsRegistry: monitoring.NewRegistry(),
				Logger:          manager.Logger,
			}
			err = inp.Run(ctx, clientCounters.BuildConnector())
		})

		cancel()
		wg.Wait()
		require.NoError(t, err)
		require.Equal(t, 0, clientCounters.Active())
	})

	t.Run("continue sending from last known position", func(t *testing.T) {
		log := logptest.NewTestingLogger(t, "test")

		type runConfig struct{ Max int }

		store := testOpenStore(t, "test", createSampleStore(t, nil))
		defer store.Release()

		manager := simpleManagerWithConfigure(t, func(cfg *conf.C, _ *logp.Logger) ([]Source, Input, error) {
			config := runConfig{}
			if err := cfg.Unpack(&config); err != nil {
				return nil, nil, err
			}

			inp := &fakeTestInput{
				OnRun: func(_ input.Context, _ Source, cursor Cursor, pub Publisher) error {
					state := struct{ N int }{}
					if !cursor.IsNew() {
						if err := cursor.Unpack(&state); err != nil {
							return fmt.Errorf("failed to unpack cursor: %w", err)
						}
					}

					for i := 0; i < config.Max; i++ {
						event := beat.Event{Fields: mapstr.M{"n": state.N}}
						state.N++
						_ = pub.Publish(event, state)
					}
					return nil
				},
			}

			return sourceList("test"), inp, nil
		})

		var ids []int
		pipeline := pubtest.ConstClient(&pubtest.FakeClient{
			PublishFunc: func(event beat.Event) {
				id, _ := event.Fields["n"].(int)
				ids = append(ids, id)
			},
		})

		// create and run first instance
		inp, err := manager.Create(conf.MustNewConfigFrom(runConfig{Max: 3}))
		require.NoError(t, err)
		id := uuid.Must(uuid.NewV4()).String()
		ctx := input.Context{
			ID:              id,
			IDWithoutName:   id,
			Name:            inp.Name(),
			Cancelation:     context.Background(),
			MetricsRegistry: monitoring.NewRegistry(),
			Logger:          log,
		}
		require.NoError(t, inp.Run(ctx, pipeline))

		// create and run second instance
		inp, err = manager.Create(conf.MustNewConfigFrom(runConfig{Max: 3}))
		require.NoError(t, err)
		id = uuid.Must(uuid.NewV4()).String()
		ctx = input.Context{
			ID:              id,
			IDWithoutName:   id,
			Name:            inp.Name(),
			Cancelation:     context.Background(),
			MetricsRegistry: monitoring.NewRegistry(),
			Logger:          log,
		}
		_ = inp.Run(ctx, pipeline)

		// verify
		assert.Equal(t, []int{0, 1, 2, 3, 4, 5}, ids)
	})

	t.Run("event ACK triggers execution of update operations", func(t *testing.T) {
		defer resources.NewGoroutinesChecker().Check(t)

		store := createSampleStore(t, nil)
		var wgSend sync.WaitGroup
		wgSend.Add(1)
		manager := constInput(t, sourceList("key"), &fakeTestInput{
			OnRun: func(ctx input.Context, _ Source, _ Cursor, pub Publisher) error {
				defer wgSend.Done()
				fields := mapstr.M{"hello": "world"}
				_ = pub.Publish(beat.Event{Fields: fields}, "test-cursor-state1")
				_ = pub.Publish(beat.Event{Fields: fields}, "test-cursor-state2")
				_ = pub.Publish(beat.Event{Fields: fields}, "test-cursor-state3")
				_ = pub.Publish(beat.Event{Fields: fields}, nil)
				_ = pub.Publish(beat.Event{Fields: fields}, "test-cursor-state4")
				_ = pub.Publish(beat.Event{Fields: fields}, "test-cursor-state5")
				_ = pub.Publish(beat.Event{Fields: fields}, "test-cursor-state6")
				return nil
			},
		})
		manager.StateStore = store

		inp, err := manager.Create(conf.NewConfig())
		require.NoError(t, err)

		cancelCtx := t.Context()

		// setup publishing pipeline and capture ACKer, so we can simulate progress in the Output
		var acker beat.EventListener
		var wgACKer sync.WaitGroup
		wgACKer.Add(1)
		pipeline := &pubtest.FakeConnector{
			ConnectFunc: func(cfg beat.ClientConfig) (beat.Client, error) {
				defer wgACKer.Done()
				acker = cfg.EventListener
				return &pubtest.FakeClient{
					PublishFunc: func(event beat.Event) {
						acker.AddEvent(event, true)
					},
				}, nil
			},
		}

		// start the input
		var wg sync.WaitGroup
		wg.Go(func() {
			id := uuid.Must(uuid.NewV4()).String()
			ctx := input.Context{
				ID:              id,
				IDWithoutName:   id,
				Name:            inp.Name(),
				Cancelation:     cancelCtx,
				MetricsRegistry: monitoring.NewRegistry(),
				Logger:          manager.Logger,
			}
			err = inp.Run(ctx, pipeline)
		})
		// wait for test setup to shut down
		defer wg.Wait()

		// wait for setup complete and events being sent (pending operations in the pipeline)
		wgACKer.Wait()
		wgSend.Wait()

		// 1. No cursor state in store yet, all operations are still pending
		require.Nil(t, store.snapshot()["test::key"].Cursor)

		// ACK first 2 events and check snapshot state
		acker.ACKEvents(2)
		require.Equal(t, "test-cursor-state2", store.snapshot()["test::key"].Cursor)

		// ACK 1 events and check snapshot state (3 events published)
		acker.ACKEvents(1)
		require.Equal(t, "test-cursor-state3", store.snapshot()["test::key"].Cursor)

		// ACK event without cursor update and check snapshot state not modified
		acker.ACKEvents(1)
		require.Equal(t, "test-cursor-state3", store.snapshot()["test::key"].Cursor)

		// ACK rest
		acker.ACKEvents(3)
		require.Equal(t, "test-cursor-state6", store.snapshot()["test::key"].Cursor)
	})
}

func TestLockResource(t *testing.T) {
	t.Run("can lock unused resource", func(t *testing.T) {
		store := testOpenStore(t, "test", createSampleStore(t, nil))
		defer store.Release()

		res := store.Get("test::key")
		err := lockResource(logptest.NewTestingLogger(t, "test"), res, context.TODO())
		require.NoError(t, err)
	})

	t.Run("fail to lock resource in use when context is cancelled", func(t *testing.T) {
		log := logptest.NewTestingLogger(t, "test")

		store := testOpenStore(t, "test", createSampleStore(t, nil))
		defer store.Release()

		resUsed := store.Get("test::key")
		err := lockResource(log, resUsed, context.TODO())
		require.NoError(t, err)

		// fail to lock resource in use
		ctx, cancel := context.WithCancel(context.TODO())
		cancel()
		resFail := store.Get("test::key")
		err = lockResource(log, resFail, ctx)
		require.Error(t, err)
		resFail.Release()

		// unlock and release resource in use -> it should be marked finished now
		releaseResource(resUsed)
		require.True(t, resUsed.Finished())
	})

	t.Run("succeed to lock resource after it has been released", func(t *testing.T) {
		log := logptest.NewTestingLogger(t, "test")

		store := testOpenStore(t, "test", createSampleStore(t, nil))
		defer store.Release()

		resUsed := store.Get("test::key")
		err := lockResource(log, resUsed, context.TODO())
		require.NoError(t, err)

		var wg sync.WaitGroup
		wg.Go(func() {
			resOther := store.Get("test::key")
			err := lockResource(log, resOther, context.TODO())
			if err == nil {
				releaseResource(resOther)
			}
		})

		go func() {
			time.Sleep(100 * time.Millisecond)
			releaseResource(resUsed)
		}()

		wg.Wait() // <- block forever if waiting go-routine can not acquire lock
	})
}

func (s stringSource) Name() string { return string(s) }

func simpleManagerWithConfigure(t *testing.T, configure func(*conf.C, *logp.Logger) ([]Source, Input, error)) *InputManager {
	return &InputManager{
		Logger:     logptest.NewTestingLogger(t, "test"),
		StateStore: createSampleStore(t, nil),
		Type:       "test",
		Configure:  configure,
	}
}

func constConfigureResult(t *testing.T, sources []Source, inp Input, err error) *InputManager {
	return simpleManagerWithConfigure(t, func(cfg *conf.C, _ *logp.Logger) ([]Source, Input, error) {
		return sources, inp, err
	})
}

func failingManager(t *testing.T, err error) *InputManager {
	return constConfigureResult(t, nil, nil, err)
}

func constInput(t *testing.T, sources []Source, inp Input) *InputManager {
	return constConfigureResult(t, sources, inp, nil)
}

func (f *fakeTestInput) Name() string { return "test" }

func (f *fakeTestInput) Test(source Source, ctx input.TestContext) error {
	if f.OnTest != nil {
		return f.OnTest(source, ctx)
	}
	return nil
}

func (f *fakeTestInput) Run(ctx input.Context, source Source, cursor Cursor, pub Publisher) error {
	if f.OnRun != nil {
		return f.OnRun(ctx, source, cursor, pub)
	}
	return nil
}

func sourceList(names ...string) []Source {
	tmp := make([]Source, len(names))
	for i, name := range names {
		tmp[i] = stringSource(name)
	}
	return tmp
}
