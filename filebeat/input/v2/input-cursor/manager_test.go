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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	input "github.com/menderesk/beats/v7/filebeat/input/v2"
	v2 "github.com/menderesk/beats/v7/filebeat/input/v2"
	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/logp"
	pubtest "github.com/menderesk/beats/v7/libbeat/publisher/testing"
	"github.com/menderesk/beats/v7/libbeat/tests/resources"
	"github.com/menderesk/go-concert/unison"
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
			Logger:              logp.NewLogger("test"),
			StateStore:          store,
			Type:                "test",
			DefaultCleanTimeout: 10 * time.Millisecond,
		}

		err := manager.Init(&grp, v2.ModeRun)
		require.NoError(t, err)

		time.Sleep(200 * time.Millisecond)
		grp.Stop()

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
		defer grp.Stop()
		manager := &InputManager{
			Logger:              logp.NewLogger("test"),
			StateStore:          store,
			Type:                "test",
			DefaultCleanTimeout: 10 * time.Millisecond,
		}

		err := manager.Init(&grp, v2.ModeRun)
		require.NoError(t, err)

		for len(store.snapshot()) > 0 {
			time.Sleep(1 * time.Millisecond)
		}
	})
}

func TestManager_Create(t *testing.T) {
	t.Run("fail if no source is configured", func(t *testing.T) {
		manager := constInput(t, nil, &fakeTestInput{})
		_, err := manager.Create(common.NewConfig())
		require.Error(t, err)
	})

	t.Run("fail if config error", func(t *testing.T) {
		manager := failingManager(t, errors.New("oops"))
		_, err := manager.Create(common.NewConfig())
		require.Error(t, err)
	})

	t.Run("fail if no input runner is returned", func(t *testing.T) {
		manager := constInput(t, sourceList("test"), nil)
		_, err := manager.Create(common.NewConfig())
		require.Error(t, err)
	})

	t.Run("configure ok", func(t *testing.T) {
		manager := constInput(t, sourceList("test"), &fakeTestInput{})
		_, err := manager.Create(common.NewConfig())
		require.NoError(t, err)
	})

	t.Run("configuring inputs with overlapping sources is allowed", func(t *testing.T) {
		manager := simpleManagerWithConfigure(t, func(cfg *common.Config) ([]Source, Input, error) {
			config := struct{ Sources []string }{}
			err := cfg.Unpack(&config)
			return sourceList(config.Sources...), &fakeTestInput{}, err
		})

		_, err := manager.Create(common.MustNewConfigFrom(map[string]interface{}{
			"sources": []string{"a"},
		}))
		require.NoError(t, err)

		_, err = manager.Create(common.MustNewConfigFrom(map[string]interface{}{
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
			OnTest: func(source Source, _ v2.TestContext) error {
				mu.Lock()
				defer mu.Unlock()
				seen = append(seen, source.Name())
				return nil
			},
		})

		inp, err := manager.Create(common.NewConfig())
		require.NoError(t, err)

		err = inp.Test(input.TestContext{})
		require.NoError(t, err)

		sort.Strings(seen)
		require.Equal(t, []string{"source1", "source2"}, seen)
	})

	t.Run("cancel gets distributed to all source tests", func(t *testing.T) {
		defer resources.NewGoroutinesChecker().Check(t)

		manager := constInput(t, sources, &fakeTestInput{
			OnTest: func(_ Source, ctx v2.TestContext) error {
				<-ctx.Cancelation.Done()
				return nil
			},
		})

		inp, err := manager.Create(common.NewConfig())
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.TODO())

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = inp.Test(input.TestContext{Cancelation: ctx})
		}()

		cancel()
		wg.Wait()
		require.NoError(t, err)
	})

	t.Run("fail if test for one source fails", func(t *testing.T) {
		defer resources.NewGoroutinesChecker().Check(t)

		failing := Source(stringSource("source1"))
		sources := []Source{failing, stringSource("source2")}

		manager := constInput(t, sources, &fakeTestInput{
			OnTest: func(source Source, _ v2.TestContext) error {
				if source == failing {
					t.Log("return error")
					return errors.New("oops")
				}
				t.Log("return ok")
				return nil
			},
		})

		inp, err := manager.Create(common.NewConfig())
		require.NoError(t, err)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = inp.Test(input.TestContext{})
			t.Logf("Test returned: %v", err)
		}()

		wg.Wait()
		require.Error(t, err)
	})

	t.Run("panic is captured", func(t *testing.T) {
		defer resources.NewGoroutinesChecker().Check(t)

		manager := constInput(t, sources, &fakeTestInput{
			OnTest: func(source Source, _ v2.TestContext) error {
				panic("oops")
			},
		})

		inp, err := manager.Create(common.NewConfig())
		require.NoError(t, err)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = inp.Test(input.TestContext{Logger: logp.NewLogger("test")})
			t.Logf("Test returned: %v", err)
		}()

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

		inp, err := manager.Create(common.NewConfig())
		require.NoError(t, err)

		cancelCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var clientCounters pubtest.ClientCounter
		err = inp.Run(v2.Context{
			Logger:      manager.Logger,
			Cancelation: cancelCtx,
		}, clientCounters.BuildConnector())
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

		inp, err := manager.Create(common.NewConfig())
		require.NoError(t, err)

		cancelCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var clientCounters pubtest.ClientCounter
		err = inp.Run(v2.Context{
			Logger:      manager.Logger,
			Cancelation: cancelCtx,
		}, clientCounters.BuildConnector())
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

		inp, err := manager.Create(common.NewConfig())
		require.NoError(t, err)

		cancelCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var clientCounters pubtest.ClientCounter
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = inp.Run(v2.Context{
				Logger:      manager.Logger,
				Cancelation: cancelCtx,
			}, clientCounters.BuildConnector())
		}()

		cancel()
		wg.Wait()
		require.NoError(t, err)
		require.Equal(t, 0, clientCounters.Active())
	})

	t.Run("continue sending from last known position", func(t *testing.T) {
		log := logp.NewLogger("test")

		type runConfig struct{ Max int }

		store := testOpenStore(t, "test", createSampleStore(t, nil))
		defer store.Release()

		manager := simpleManagerWithConfigure(t, func(cfg *common.Config) ([]Source, Input, error) {
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
						event := beat.Event{Fields: common.MapStr{"n": state.N}}
						state.N++
						pub.Publish(event, state)
					}
					return nil
				},
			}

			return sourceList("test"), inp, nil
		})

		var ids []int
		pipeline := pubtest.ConstClient(&pubtest.FakeClient{
			PublishFunc: func(event beat.Event) {
				id := event.Fields["n"].(int)
				ids = append(ids, id)
			},
		})

		// create and run first instance
		inp, err := manager.Create(common.MustNewConfigFrom(runConfig{Max: 3}))
		require.NoError(t, err)
		require.NoError(t, inp.Run(input.Context{
			Logger:      log,
			Cancelation: context.Background(),
		}, pipeline))

		// create and run second instance instance
		inp, err = manager.Create(common.MustNewConfigFrom(runConfig{Max: 3}))
		require.NoError(t, err)
		inp.Run(input.Context{
			Logger:      log,
			Cancelation: context.Background(),
		}, pipeline)

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
				fields := common.MapStr{"hello": "world"}
				pub.Publish(beat.Event{Fields: fields}, "test-cursor-state1")
				pub.Publish(beat.Event{Fields: fields}, "test-cursor-state2")
				pub.Publish(beat.Event{Fields: fields}, "test-cursor-state3")
				pub.Publish(beat.Event{Fields: fields}, nil)
				pub.Publish(beat.Event{Fields: fields}, "test-cursor-state4")
				pub.Publish(beat.Event{Fields: fields}, "test-cursor-state5")
				pub.Publish(beat.Event{Fields: fields}, "test-cursor-state6")
				return nil
			},
		})
		manager.StateStore = store

		inp, err := manager.Create(common.NewConfig())
		require.NoError(t, err)

		cancelCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// setup publishing pipeline and capture ACKer, so we can simulate progress in the Output
		var acker beat.ACKer
		var wgACKer sync.WaitGroup
		wgACKer.Add(1)
		pipeline := &pubtest.FakeConnector{
			ConnectFunc: func(cfg beat.ClientConfig) (beat.Client, error) {
				defer wgACKer.Done()
				acker = cfg.ACKHandler
				return &pubtest.FakeClient{
					PublishFunc: func(event beat.Event) {
						acker.AddEvent(event, true)
					},
				}, nil
			},
		}

		// start the input
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = inp.Run(v2.Context{
				Logger:      manager.Logger,
				Cancelation: cancelCtx,
			}, pipeline)
		}()
		// wait for test setup to shutdown
		defer wg.Wait()

		// wait for setup complete and events being send (pending operations in the pipeline)
		wgACKer.Wait()
		wgSend.Wait()

		// 1. No cursor state in store yet, all operations are still pending
		require.Equal(t, nil, store.snapshot()["test::key"].Cursor)

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
		err := lockResource(logp.NewLogger("test"), res, context.TODO())
		require.NoError(t, err)
	})

	t.Run("fail to lock resource in use when context is cancelled", func(t *testing.T) {
		log := logp.NewLogger("test")

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
		log := logp.NewLogger("test")

		store := testOpenStore(t, "test", createSampleStore(t, nil))
		defer store.Release()

		resUsed := store.Get("test::key")
		err := lockResource(log, resUsed, context.TODO())
		require.NoError(t, err)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			resOther := store.Get("test::key")
			err := lockResource(log, resOther, context.TODO())
			if err == nil {
				releaseResource(resOther)
			}
		}()

		go func() {
			time.Sleep(100 * time.Millisecond)
			releaseResource(resUsed)
		}()

		wg.Wait() // <- block forever if waiting go-routine can not acquire lock
	})
}

func (s stringSource) Name() string { return string(s) }

func simpleManagerWithConfigure(t *testing.T, configure func(*common.Config) ([]Source, Input, error)) *InputManager {
	return &InputManager{
		Logger:     logp.NewLogger("test"),
		StateStore: createSampleStore(t, nil),
		Type:       "test",
		Configure:  configure,
	}
}

func constConfigureResult(t *testing.T, sources []Source, inp Input, err error) *InputManager {
	return simpleManagerWithConfigure(t, func(cfg *common.Config) ([]Source, Input, error) {
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
