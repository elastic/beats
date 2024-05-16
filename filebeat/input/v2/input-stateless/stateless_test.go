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

package stateless_test

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	pubtest "github.com/elastic/beats/v7/libbeat/publisher/testing"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type fakeStatelessInput struct {
	OnTest func(v2.TestContext) error
	OnRun  func(v2.Context, stateless.Publisher) error
}

func TestStateless_Run(t *testing.T) {
	t.Run("events are published", func(t *testing.T) {
		const numEvents = 5

		ch := make(chan beat.Event)
		connector := pubtest.ConstClient(pubtest.ChClient(ch))

		input := createConfiguredInput(t, constInputManager(&fakeStatelessInput{
			OnRun: func(ctx v2.Context, publisher stateless.Publisher) error {
				defer close(ch)
				for i := 0; i < numEvents; i++ {
					publisher.Publish(beat.Event{Fields: map[string]interface{}{"id": i}})
				}
				return nil
			},
		}), nil)

		var err error
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = input.Run(v2.Context{}, connector)
		}()

		var receivedEvents int
		for range ch {
			receivedEvents++
		}
		wg.Wait()

		require.NoError(t, err)
		require.Equal(t, numEvents, receivedEvents)
	})

	t.Run("capture panic and return error", func(t *testing.T) {
		input := createConfiguredInput(t, constInputManager(&fakeStatelessInput{
			OnRun: func(_ v2.Context, _ stateless.Publisher) error {
				panic("oops")
			},
		}), nil)

		var clientCounters pubtest.ClientCounter
		err := input.Run(v2.Context{}, clientCounters.BuildConnector())

		require.Error(t, err)
		require.Equal(t, 1, clientCounters.Total())
		require.Equal(t, 0, clientCounters.Active())
	})

	t.Run("publisher unblocks if shutdown signal is send", func(t *testing.T) {
		// the input blocks in the publisher. We loop until the shutdown signal is received
		var started atomic.Bool
		input := createConfiguredInput(t, constInputManager(&fakeStatelessInput{
			OnRun: func(ctx v2.Context, publisher stateless.Publisher) error {
				for ctx.Cancelation.Err() == nil {
					started.Store(true)
					publisher.Publish(beat.Event{
						Fields: mapstr.M{
							"hello": "world",
						},
					})
				}
				return ctx.Cancelation.Err()
			},
		}), nil)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// connector creates a client the blocks forever until the shutdown signal is received
		var publishCalls atomic.Int
		connector := pubtest.FakeConnector{
			ConnectFunc: func(config beat.ClientConfig) (beat.Client, error) {
				return &pubtest.FakeClient{
					PublishFunc: func(event beat.Event) {
						publishCalls.Inc()
						// Unlock Publish once the input has been cancelled
						<-ctx.Done()
					},
				}, nil
			},
		}

		var wg sync.WaitGroup
		var err error
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = input.Run(v2.Context{Cancelation: ctx}, connector)
		}()

		// signal and wait for shutdown
		for !started.Load() {
			runtime.Gosched()
		}
		cancel()
		wg.Wait()

		// validate
		require.Equal(t, context.Canceled, err)
		require.Equal(t, 1, publishCalls.Load())
	})

	t.Run("do not start input of pipeline connection fails", func(t *testing.T) {
		errOpps := errors.New("oops")
		connector := pubtest.FailingConnector(errOpps)

		var run atomic.Int
		input := createConfiguredInput(t, constInputManager(&fakeStatelessInput{
			OnRun: func(_ v2.Context, publisher stateless.Publisher) error {
				run.Inc()
				return nil
			},
		}), nil)

		err := input.Run(v2.Context{}, connector)
		require.True(t, errors.Is(err, errOpps))
		require.Equal(t, 0, run.Load())
	})
}

func (f *fakeStatelessInput) Name() string { return "test" }

func (f *fakeStatelessInput) Test(ctx v2.TestContext) error {
	if f.OnTest != nil {
		return f.OnTest(ctx)
	}
	return nil
}

func (f *fakeStatelessInput) Run(ctx v2.Context, publish stateless.Publisher) error {
	if f.OnRun != nil {
		return f.OnRun(ctx, publish)
	}
	return errors.New("oops, run not implemented")
}

func createConfiguredInput(t *testing.T, manager stateless.InputManager, config map[string]interface{}) v2.Input {
	input, err := manager.Create(conf.MustNewConfigFrom(config))
	require.NoError(t, err)
	return input
}

func constInputManager(input stateless.Input) stateless.InputManager {
	return stateless.NewInputManager(constInput(input))
}

func constInput(input stateless.Input) func(*conf.C) (stateless.Input, error) {
	return func(_ *conf.C) (stateless.Input, error) {
		return input, nil
	}
}
