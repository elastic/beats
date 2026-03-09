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

package autodiscover

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/tests/resources"
	"github.com/elastic/elastic-agent-autodiscover/bus"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/keystore"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type mockRunner struct {
	mutex            sync.Mutex
	config           *conf.C
	started, stopped bool
}

func (m *mockRunner) Start() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.started = true
	m.stopped = false
}
func (m *mockRunner) Stop() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.stopped = true
	m.started = false
}

func (m *mockRunner) Clone() *mockRunner {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return &mockRunner{
		config:  m.config,
		started: m.started,
		stopped: m.stopped,
	}
}
func (m *mockRunner) String() string {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	out := mapstr.M{}
	m.config.Unpack(&out) //nolint:errcheck // This is a test file
	return fmt.Sprintf("config: %v, started=%v, stopped=%v", out.String(), m.started, m.stopped)
}

type mockAdapter struct {
	mutex   sync.Mutex
	configs []*conf.C
	runners []*mockRunner

	CheckConfigCallCount int
}

// CreateConfig generates a valid list of configs from the given event, the received event will have all keys defined by `StartFilter`
func (m *mockAdapter) CreateConfig(event bus.Event) ([]*conf.C, error) {
	if cfgs, ok := event["config"]; ok {
		if confs, ok := cfgs.([]*conf.C); ok {
			return confs, nil
		}

		return nil, fmt.Errorf("event['config'] is of type '%T', expecting '[]*conf.C'", cfgs)
	}

	return m.configs, nil
}

// CheckConfig tests given config to check if it will work or not, returns errors in case it won't work
func (m *mockAdapter) CheckConfig(c *conf.C) error {
	m.CheckConfigCallCount++

	config := struct {
		Broken bool `config:"broken"`
	}{}
	if err := c.Unpack(&config); err != nil {
		return fmt.Errorf("cannot unpack config: %w", err)
	}

	if config.Broken {
		return fmt.Errorf("Broken config")
	}

	return nil
}

// Create returns a mockRunner with the provided config. If
// the config contains `err_non_reloadable: true`, then a
// common.ErrNonReloadable is returned alongside a nil runner.
func (m *mockAdapter) Create(_ beat.PipelineConnector, config *conf.C) (cfgfile.Runner, error) {
	// On error false is returned, that's enough to keep a correct behaviour
	nonReloadable, _ := config.Bool("err_non_reloadable", -1)
	if nonReloadable {
		return nil, common.ErrNonReloadable{
			Err: errors.New("a non reloadable error"),
		}
	}

	runner := &mockRunner{
		config: config,
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.runners = append(m.runners, runner)
	return runner, nil
}

func (m *mockAdapter) Runners() []*mockRunner {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	res := make([]*mockRunner, 0, len(m.runners))
	for _, r := range m.runners {
		res = append(res, r.Clone())
	}
	return res
}

func (m *mockAdapter) EventFilter() []string {
	return []string{"meta"}
}

type mockProvider struct{}

// Start the autodiscover process, send all configured events to the bus
func (d *mockProvider) Start() {}

// Stop the autodiscover process
func (d *mockProvider) Stop() {}

func (d *mockProvider) String() string {
	return "mock"
}

func TestNilAutodiscover(t *testing.T) {
	var autodiscover *Autodiscover
	autodiscover.Start()
	autodiscover.Stop()
}

func TestAutodiscover(t *testing.T) {
	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	// Register mock autodiscover provider
	busChan := make(chan bus.Bus, 1)
	Registry = NewRegistry()
	err := Registry.AddProvider("mock",
		func(beatName string, b bus.Bus, uuid uuid.UUID, c *conf.C, k keystore.Keystore, l *logp.Logger) (Provider, error) {
			// intercept bus to mock events
			busChan <- b

			return &mockProvider{}, nil
		})
	if err != nil {
		t.Fatalf("cannot add provider to registry: %s", err)
	}

	// Create a mock adapter
	runnerConfig, _ := conf.NewConfigFrom(map[string]string{
		"runner": "1",
	})
	adapter := mockAdapter{
		configs: []*conf.C{runnerConfig},
	}

	// and settings:
	providerConfig, _ := conf.NewConfigFrom(map[string]string{
		"type": "mock",
	})
	config := Config{
		Providers: []*conf.C{providerConfig},
	}
	k, _ := keystore.NewFileKeystore("test")
	// Create autodiscover manager
	logger := logptest.NewTestingLogger(t, "")
	autodiscover, err := NewAutodiscover("test", nil, &adapter, &adapter, &config, k, logger)
	if err != nil {
		t.Fatal(err)
	}

	// set the debounce period to something small in order to
	// speed up the tests. This seems to be the sweet stop
	// for the fastest test run
	autodiscover.debouncePeriod = 99 * time.Millisecond

	// Start it
	autodiscover.Start()
	defer autodiscover.Stop()
	eventBus := <-busChan

	// Test start event
	// This start event will trigger an input reload
	t.Log("Sending first start event, there will be 1 runner running")
	eventBus.Publish(bus.Event{
		"id":       "foo",
		"provider": "mock",
		"start":    true,
		"meta": mapstr.M{
			"foo": "bar",
		},
	})

	requireRunningRunners(t, autodiscover, 1)
	runners := adapter.Runners()
	require.Len(t, runners, 1)
	require.Len(t, autodiscover.configs["mock:foo"], 1)
	require.True(t, runners[0].started)
	require.False(t, runners[0].stopped)

	// Test update
	// Autodiscover will not call "Reload" because the input
	// is already running. This will not trigger an input reload
	t.Log("Seeding first 'update' event, there will be 1 runner running")
	eventBus.Publish(bus.Event{
		"id":       "foo",
		"provider": "mock",
		"start":    true,
		"meta": mapstr.M{
			"foo": "baz",
		},
	})

	requireRunningRunners(t, autodiscover, 1)
	runners = adapter.Runners()
	require.Len(t, runners, 1)
	require.Len(t, autodiscover.configs["mock:foo"], 1)
	require.True(t, runners[0].started)
	require.False(t, runners[0].stopped)

	// Test stop/start
	// This stop event will trigger an input Reload
	t.Log("Seeding first stop event, there will be 0 runners running")
	eventBus.Publish(bus.Event{
		"id":       "foo",
		"provider": "mock",
		"stop":     true,
		"meta": mapstr.M{
			"foo": "baz",
		},
	})

	requireRunningRunners(t, autodiscover, 0)

	// This start event will trigger an input reload
	t.Log("Sending second start event, there will be 1 runner running")
	eventBus.Publish(bus.Event{
		"id":       "foo",
		"provider": "mock",
		"start":    true,
		"meta": mapstr.M{
			"foo": "baz",
		},
	})

	requireRunningRunners(t, autodiscover, 1)
	runners = adapter.Runners()
	require.Len(t, runners, 2)
	require.Len(t, autodiscover.configs["mock:foo"], 1)
	require.True(t, runners[0].stopped)
	require.True(t, runners[1].started)
	require.False(t, runners[1].stopped)

	// Test stop event
	// This stop event will trigger an input reload
	t.Log("sending second stop event, there will be 0 runners running")
	eventBus.Publish(bus.Event{
		"id":       "foo",
		"provider": "mock",
		"stop":     true,
		"meta": mapstr.M{
			"foo": "baz",
		},
	})

	// Instead of ensuring the number of running runners, ensure
	// that the second started runner has stopped
	require.Eventually(t,
		func() bool {
			return adapter.Runners()[1].stopped
		},
		10*time.Second,
		100*time.Millisecond,
		"adapter.Runners()[1] has not stopped")

	runners = adapter.Runners()
	require.Len(t, runners, 2)
	require.Empty(t, autodiscover.configs["mock:foo"])
	require.False(t, runners[1].started)
	require.True(t, runners[1].stopped)
}

func TestAutodiscoverHash(t *testing.T) {
	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	// Register mock autodiscover provider
	busChan := make(chan bus.Bus, 1)

	Registry = NewRegistry()
	err := Registry.AddProvider("mock", func(beatName string, b bus.Bus, uuid uuid.UUID, c *conf.C, k keystore.Keystore, l *logp.Logger) (Provider, error) {
		// intercept bus to mock events
		busChan <- b

		return &mockProvider{}, nil
	})
	if err != nil {
		t.Fatalf("cannot add provider to registry: %s", err)
	}

	// Create a mock adapter
	runnerConfig1, _ := conf.NewConfigFrom(map[string]string{
		"runner": "1",
	})
	runnerConfig2, _ := conf.NewConfigFrom(map[string]string{
		"runner": "2",
	})
	adapter := mockAdapter{
		configs: []*conf.C{runnerConfig1, runnerConfig2},
	}

	// and settings:
	providerConfig, _ := conf.NewConfigFrom(map[string]string{
		"type": "mock",
	})
	config := Config{
		Providers: []*conf.C{providerConfig},
	}
	k, _ := keystore.NewFileKeystore("test")
	logger := logptest.NewTestingLogger(t, "")
	// Create autodiscover manager
	autodiscover, err := NewAutodiscover("test", nil, &adapter, &adapter, &config, k, logger)
	if err != nil {
		t.Fatal(err)
	}

	// Start it
	autodiscover.Start()
	defer autodiscover.Stop()
	eventBus := <-busChan

	// Test start event
	eventBus.Publish(bus.Event{
		"id":       "foo",
		"provider": "mock",
		"start":    true,
		"meta": mapstr.M{
			"foo": "bar",
		},
	})
	wait(t, func() bool { return len(adapter.Runners()) == 2 })

	runners := adapter.Runners()
	assert.Len(t, runners, 2)
	assert.Len(t, autodiscover.configs["mock:foo"], 2)
	assert.True(t, runners[0].started)
	assert.False(t, runners[0].stopped)
	assert.True(t, runners[1].started)
	assert.False(t, runners[1].stopped)
}

func TestAutodiscoverDuplicatedConfigConfigCheckCalledOnce(t *testing.T) {
	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	// Register mock autodiscover provider
	busChan := make(chan bus.Bus, 1)

	Registry = NewRegistry()
	err := Registry.AddProvider("mock", func(beatName string, b bus.Bus, uuid uuid.UUID, c *conf.C, k keystore.Keystore, l *logp.Logger) (Provider, error) {
		// intercept bus to mock events
		busChan <- b

		return &mockProvider{}, nil
	})
	if err != nil {
		t.Fatalf("cannot add provider to registry: %s", err)
	}
	// Create a mock adapter that returns a duplicated config
	runnerConfig, _ := conf.NewConfigFrom(map[string]string{
		"id": "foo",
	})
	adapter := mockAdapter{
		configs: []*conf.C{runnerConfig, runnerConfig},
	}

	// and settings:
	providerConfig, _ := conf.NewConfigFrom(map[string]string{
		"type": "mock",
	})
	config := Config{
		Providers: []*conf.C{providerConfig},
	}
	k, _ := keystore.NewFileKeystore("test")
	logger := logptest.NewTestingLogger(t, "")
	// Create autodiscover manager
	autodiscover, err := NewAutodiscover("test", nil, &adapter, &adapter, &config, k, logger)
	if err != nil {
		t.Fatal(err)
	}

	// Start it
	autodiscover.Start()
	defer autodiscover.Stop()
	eventBus := <-busChan

	// Publish a couple of events.
	for i := 0; i < 2; i++ {
		eventBus.Publish(bus.Event{
			"id":       "foo",
			"provider": "mock",
			"start":    true,
			"meta": mapstr.M{
				"foo": "bar",
			},
		})

		assert.Eventually(t, func() bool {
			return len(adapter.Runners()) == 1
		}, 10*time.Second, 100*time.Millisecond, "Only one runner should be started")

		assert.Eventually(t, func() bool {
			return adapter.CheckConfigCallCount == 1
		}, 10*time.Second, 100*time.Millisecond, "Check config should have been called only once")
	}
}

func TestAutodiscoverWithConfigCheckFailures(t *testing.T) {
	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	// Register mock autodiscover provider
	busChan := make(chan bus.Bus, 1)
	Registry = NewRegistry()
	err := Registry.AddProvider("mock", func(beatName string, b bus.Bus, uuid uuid.UUID, c *conf.C, k keystore.Keystore, l *logp.Logger) (Provider, error) {
		// intercept bus to mock events
		busChan <- b

		return &mockProvider{}, nil
	})
	if err != nil {
		t.Fatalf("cannot add provider to registry: %s", err)
	}

	// Create a mock adapter
	runnerConfig1, _ := conf.NewConfigFrom(map[string]string{
		"broken": "true",
	})
	runnerConfig2, _ := conf.NewConfigFrom(map[string]string{
		"runner": "2",
	})
	adapter := mockAdapter{
		configs: []*conf.C{runnerConfig1, runnerConfig2},
	}

	// and settings:
	providerConfig, _ := conf.NewConfigFrom(map[string]string{
		"type": "mock",
	})
	config := Config{
		Providers: []*conf.C{providerConfig},
	}
	k, _ := keystore.NewFileKeystore("test")
	logger := logptest.NewTestingLogger(t, "")
	// Create autodiscover manager
	autodiscover, err := NewAutodiscover("test", nil, &adapter, &adapter, &config, k, logger)
	if err != nil {
		t.Fatal(err)
	}

	// Start it
	autodiscover.Start()
	defer autodiscover.Stop()
	eventBus := <-busChan

	// Test start event
	eventBus.Publish(bus.Event{
		"id":       "foo",
		"provider": "mock",
		"start":    true,
		"meta": mapstr.M{
			"foo": "bar",
		},
	})

	// As only the second config is valid, total runners will be 1
	wait(t, func() bool { return len(adapter.Runners()) == 1 })
	assert.Len(t, autodiscover.configs["mock:foo"], 1)
}

func TestAutodiscoverWithMutlipleEntries(t *testing.T) {
	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	// Register mock autodiscover provider
	busChan := make(chan bus.Bus, 1)
	Registry = NewRegistry()
	err := Registry.AddProvider("mock", func(beatName string, b bus.Bus, uuid uuid.UUID, c *conf.C, k keystore.Keystore, l *logp.Logger) (Provider, error) {
		// intercept bus to mock events
		busChan <- b

		return &mockProvider{}, nil
	})
	if err != nil {
		t.Fatalf("cannot add provider to registry: %s", err)
	}

	// Create a mock adapter
	runnerConfig, _ := conf.NewConfigFrom(map[string]string{
		"runner": "1",
	})
	adapter := mockAdapter{
		configs: []*conf.C{runnerConfig},
	}

	// and settings:
	providerConfig, _ := conf.NewConfigFrom(map[string]string{
		"type": "mock",
	})
	config := Config{
		Providers: []*conf.C{providerConfig},
	}
	k, _ := keystore.NewFileKeystore("test")
	logger := logptest.NewTestingLogger(t, "")
	// Create autodiscover manager
	autodiscover, err := NewAutodiscover("test", nil, &adapter, &adapter, &config, k, logger)
	if err != nil {
		t.Fatal(err)
	}

	// Start it
	autodiscover.Start()
	defer autodiscover.Stop()
	eventBus := <-busChan

	// Test start event
	eventBus.Publish(bus.Event{
		"id":       "foo",
		"provider": "mock",
		"start":    true,
		"meta": mapstr.M{
			"foo": "bar",
		},
		"config": []*conf.C{
			conf.MustNewConfigFrom(map[string]interface{}{
				"a": "b",
			}),
			conf.MustNewConfigFrom(map[string]interface{}{
				"x": "y",
			}),
		},
	})
	wait(t, func() bool { return len(adapter.Runners()) == 2 })

	runners := adapter.Runners()
	assert.Len(t, runners, 2)
	assert.Len(t, autodiscover.configs["mock:foo"], 2)
	check(t, runners, conf.MustNewConfigFrom(map[string]interface{}{"x": "y"}), true, false)
	check(t, runners, conf.MustNewConfigFrom(map[string]interface{}{"a": "b"}), true, false)
	// Test start event with changed configurations
	eventBus.Publish(bus.Event{
		"id":       "foo",
		"provider": "mock",
		"start":    true,
		"meta": mapstr.M{
			"foo": "bar",
		},
		"config": []*conf.C{
			conf.MustNewConfigFrom(map[string]interface{}{
				"a": "b",
			}),
			conf.MustNewConfigFrom(map[string]interface{}{
				"x": "c",
			}),
		},
	})
	wait(t, func() bool { return len(adapter.Runners()) == 3 })
	runners = adapter.Runners()
	// Ensure the first config is the same as before
	t.Log(runners)
	assert.Len(t, runners, 3)
	assert.Len(t, autodiscover.configs["mock:foo"], 2)
	check(t, runners, conf.MustNewConfigFrom(map[string]interface{}{"a": "b"}), true, false)

	// Ensure that the runner for the stale config is stopped
	wait(t, func() bool {
		check(t, adapter.Runners(), conf.MustNewConfigFrom(map[string]interface{}{"x": "c"}), true, false)
		return true
	})

	// Ensure that the new runner is started
	check(t, runners, conf.MustNewConfigFrom(map[string]interface{}{"x": "c"}), true, false)

	// Stop all the configs
	eventBus.Publish(bus.Event{
		"id":       "foo",
		"provider": "mock",
		"stop":     true,
		"meta": mapstr.M{
			"foo": "bar",
		},
		"config": []*conf.C{
			conf.MustNewConfigFrom(map[string]interface{}{
				"a": "b",
			}),
			conf.MustNewConfigFrom(map[string]interface{}{
				"x": "c",
			}),
		},
	})

	wait(t, func() bool { return adapter.Runners()[2].stopped == true })
	runners = adapter.Runners()
	check(t, runners, conf.MustNewConfigFrom(map[string]interface{}{"x": "c"}), false, true)
}

func TestAutodiscoverDebounce(t *testing.T) {
	// Register mock autodiscover provider
	busChan := make(chan bus.Bus, 1)
	Registry = NewRegistry()
	err := Registry.AddProvider("mock", func(beatName string, b bus.Bus, uuid uuid.UUID, c *conf.C, k keystore.Keystore, l *logp.Logger) (Provider, error) {
		// intercept bus to mock events
		busChan <- b

		return &mockProvider{}, nil
	})
	if err != nil {
		t.Fatalf("cannot add provider to registry: %s", err)
	}

	providerConfig, _ := conf.NewConfigFrom(map[string]string{
		"type": "mock",
	})
	config := Config{
		Providers: []*conf.C{providerConfig},
	}
	k, _ := keystore.NewFileKeystore("test")

	adapter := mockAdapter{}
	logger := logptest.NewTestingLogger(t, "")
	// Create autodiscover manager
	autodiscover, err := NewAutodiscover("test", nil, &adapter, &adapter, &config, k, logger)
	if err != nil {
		t.Fatal(err)
	}

	// set the debounce period to something small in order to
	// speed up the tests. This seems to be the sweet stop
	// for the fastest test run
	autodiscover.debouncePeriod = 99 * time.Millisecond

	// Start it
	autodiscover.Start()
	t.Cleanup(autodiscover.Stop)

	eventBus := <-busChan

	cfg, err := conf.NewConfigFrom(map[string]string{
		"foo": "bar",
	})
	if err != nil {
		t.Fatalf("error creating input config: %s", err)
	}

	// Send an event with config,
	// `Autodiscover.handleStart` will return true and
	// updated will be set to true
	eventBus.Publish(bus.Event{
		"id":       "foo",
		"provider": "mock",
		"start":    true,
		"meta": mapstr.M{
			"foo": "bar",
		},
		"config": []*conf.C{cfg},
	})

	// Send the second event without a config
	// `Autodiscover.handleStart` will return false and
	// updated must not be changed
	eventBus.Publish(bus.Event{
		"id":       "second,",
		"provider": "mock",
		"start":    true,
		"meta": mapstr.M{
			"foo": "bar",
		},
	})

	// Ensure the input has been started
	requireRunningRunners(t, autodiscover, 1)

	// Repeat the process, but this time with a stop event.
	// The same logic applies, but now we're testing the branch that calls
	// `Autodiscover.handleStop`
	eventBus.Publish(bus.Event{
		"id":       "foo",
		"provider": "mock",
		"stop":     true,
		"meta": mapstr.M{
			"foo": "bar",
		},
		"config": []*conf.C{cfg},
	})
	eventBus.Publish(bus.Event{
		"id":       "second,",
		"provider": "mock",
		"stop":     true,
		"meta": mapstr.M{
			"foo": "bar",
		},
	})
	requireRunningRunners(t, autodiscover, 0)
}

func requireRunningRunners(t *testing.T, autodiscover *Autodiscover, nRunners int) {
	t.Helper()
	nRunnersStr := strings.Builder{}
	require.Eventuallyf(t,
		func() bool {
			nRunnersStr.Reset()
			lenRunners := len(autodiscover.runners.Runners())
			fmt.Fprint(&nRunnersStr, lenRunners)
			return lenRunners == nRunners
		},
		30*time.Second,
		100*time.Millisecond,
		"expecting %d runners, got %s", nRunners, &nRunnersStr)
}

func wait(t *testing.T, test func() bool) {
	t.Helper()
	sleep := 20 * time.Millisecond
	ready := test()
	for !ready && sleep < 10*time.Second {
		time.Sleep(sleep)
		sleep = sleep + 1*time.Second
		ready = test()
	}

	if !ready {
		t.Fatal("Waiting for condition")
	}
}

func check(t *testing.T, runners []*mockRunner, expected *conf.C, started, stopped bool) {
	t.Helper()
	for _, r := range runners {
		if reflect.DeepEqual(expected, r.config) {
			ok1 := assert.Equal(t, started, r.started)
			ok2 := assert.Equal(t, stopped, r.stopped)

			if ok1 && ok2 {
				return
			}
		}
	}

	// Fail the test case if the check fails
	out := mapstr.M{}
	if err := expected.Unpack(&out); err != nil {
		t.Fatalf("cannot unpack 'out' as 'mapstr.M', err: %s", err)
	}
	t.Fatalf("expected cfg %v to be started=%v stopped=%v but have %v", out, started, stopped, runners)
}

func TestErrNonReloadableIsNotRetried(t *testing.T) {
	// Register mock autodiscover provider
	busChan := make(chan bus.Bus, 1)
	Registry = NewRegistry()
	err := Registry.AddProvider(
		"mock",
		func(beatName string,
			b bus.Bus,
			uuid uuid.UUID,
			c *conf.C,
			k keystore.Keystore,
			l *logp.Logger) (Provider, error) {

			// intercept bus to mock events
			busChan <- b

			return &mockProvider{}, nil
		})
	if err != nil {
		t.Fatalf("cannot add provider to registry: %s", err)
	}

	// Create a mock adapter, 'err_non_reloadable' will make its Create method
	// to return a common.ErrNonReloadable.
	adapter := mockAdapter{
		configs: []*conf.C{
			conf.MustNewConfigFrom(map[string]any{
				"err_non_reloadable": true,
			}),
		},
	}

	// and settings:
	providerConfig, _ := conf.NewConfigFrom(map[string]string{
		"type": "mock",
	})
	config := Config{
		Providers: []*conf.C{providerConfig},
	}
	k, _ := keystore.NewFileKeystore(filepath.Join(t.TempDir(), "keystore"))
	logger, observedLogs := logptest.NewTestingLoggerWithObserver(t, "")
	// Create autodiscover manager
	autodiscover, err := NewAutodiscover("test", nil, &adapter, &adapter, &config, k, logger)
	if err != nil {
		t.Fatal(err)
	}

	// set the debounce period to something small in order to
	// speed up the tests. This seems to be the sweet stop
	// for the fastest test run
	autodiscover.debouncePeriod = time.Millisecond

	// Start it
	autodiscover.Start()
	defer autodiscover.Stop()
	eventBus := <-busChan

	// Send an event to the bus, the event itself is not important
	// because the mockAdapter will return the same configs regardless
	// of the event
	eventBus.Publish(bus.Event{
		// That's used in the last assertion, the config key is
		// <provider name>:<id>
		"id":       "foo",
		"provider": "mock",
		"start":    true,
		"meta": mapstr.M{
			"test_name": t.Name(),
		},
	})

	// Ensure we logged the error about not retrying reloading input
	require.Eventually(
		t,
		func() bool {
			logs := observedLogs.TakeAll()
			for _, log := range logs {
				if log.Message == "all new inputs failed to start with a non-retriable error" && log.ContextMap()["error"] == "Error creating runner from config: ErrNonReloadable: a non reloadable error" {
					return true
				}
			}
			return false
		},
		time.Second*10,
		time.Millisecond*10,
		"foo error")

	// Ensure nothing is running
	requireRunningRunners(t, autodiscover, 0)
	runners := adapter.Runners()
	require.Empty(t, runners)

	// Ensure the autodiscover got the config
	require.Len(t, autodiscover.configs["mock:foo"], 1)
}

// TestAutodiscoverMetadataCleanup tests that the worker properly cleans up metadata
// for configurations that are no longer active.
func TestAutodiscoverMetadataCleanup(t *testing.T) {
	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	busChan := make(chan bus.Bus, 1)
	Registry = NewRegistry()
	err := Registry.AddProvider("mock", func(beatName string, b bus.Bus, uuid uuid.UUID, c *conf.C, k keystore.Keystore, l *logp.Logger) (Provider, error) {
		busChan <- b
		return &mockProvider{}, nil
	})
	require.NoError(t, err)

	adapter := mockAdapter{}
	providerConfig, _ := conf.NewConfigFrom(map[string]string{
		"type": "mock",
	})
	config := Config{
		Providers: []*conf.C{providerConfig},
	}
	k, _ := keystore.NewFileKeystore("test")
	logger := logptest.NewTestingLogger(t, "")

	autodiscover, err := NewAutodiscover("test", nil, &adapter, &adapter, &config, k, logger)
	require.NoError(t, err)

	autodiscover.debouncePeriod = 50 * time.Millisecond

	autodiscover.Start()
	defer autodiscover.Stop()
	eventBus := <-busChan

	fooConfig1, _ := conf.NewConfigFrom(map[string]string{
		"id": "foo-1",
	})
	fooConfig2, _ := conf.NewConfigFrom(map[string]string{
		"id": "foo-2",
	})

	// Publish event, this should create metadata entries
	eventBus.Publish(bus.Event{
		"id":       "foo",
		"provider": "mock",
		"start":    true,
		"meta": mapstr.M{
			"service":   "foo-service",
			"namespace": "test-ns",
		},
		"config": []*conf.C{fooConfig1, fooConfig2},
	})

	// Wait for configs to be processed
	wait(t, func() bool { return len(adapter.Runners()) == 2 })

	// check that configs and metadata exist
	assert.Len(t, autodiscover.configs["mock:foo"], 2)
	metaKeys := autodiscover.meta.Keys()
	assert.Len(t, metaKeys, 2, "Should have 2 metadata entries for id foo")

	// create another service "bar" with 2 configs
	barConfig1, _ := conf.NewConfigFrom(map[string]string{
		"id": "bar-1",
	})
	barConfig2, _ := conf.NewConfigFrom(map[string]string{
		"id": "bar-2",
	})
	eventBus.Publish(bus.Event{
		"id":       "bar",
		"provider": "mock",
		"start":    true,
		"meta": mapstr.M{
			"service":   "bar-service",
			"namespace": "test-ns",
		},
		"config": []*conf.C{barConfig1, barConfig2},
	})
	// Wait for configs to be processed
	wait(t, func() bool { return len(adapter.Runners()) == 4 })
	assert.Len(t, autodiscover.configs["mock:foo"], 2)
	assert.Len(t, autodiscover.configs["mock:bar"], 2)
	metaKeys = autodiscover.meta.Keys()
	assert.Len(t, metaKeys, 4, "Should have 4 metadata entries total")

	// Stop first config
	eventBus.Publish(bus.Event{
		"id":       "foo",
		"provider": "mock",
		"stop":     true,
		"meta": mapstr.M{
			"service":   "foo-service",
			"namespace": "test-ns",
		},
	})

	// Wait for some configs to be removed from active configs
	wait(t, func() bool {
		return len(autodiscover.configs["mock:foo"]) == 0 && len(autodiscover.configs["mock:bar"]) == 2
	})

	// Metadata should still exist right after stopping the config
	metaKeys = autodiscover.meta.Keys()
	assert.Len(t, metaKeys, 4, "Should still have 4 metadata entries before cleanup")

	// Wait for debounce period so the worker can run the metadata GC
	wait(t, func() bool {
		metaKeys := autodiscover.meta.Keys()
		return len(metaKeys) == 2 // Only metadata for "bar" should remain
	})

	// Stop "bar" and check for full cleanup
	eventBus.Publish(bus.Event{
		"id":       "bar",
		"provider": "mock",
		"stop":     true,
		"meta": mapstr.M{
			"service":   "bar-service",
			"namespace": "test-ns",
		},
	})

	// Wait for all configs to be removed
	wait(t, func() bool {
		return len(autodiscover.configs["mock:bar"]) == 0
	})

	// Without active configs, metadata should be cleaned up
	wait(t, func() bool {
		metaKeys := autodiscover.meta.Keys()
		return len(metaKeys) == 0
	})
}
