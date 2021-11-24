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
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/bus"
	"github.com/elastic/beats/v7/libbeat/keystore"
	"github.com/elastic/beats/v7/libbeat/tests/resources"
)

type mockRunner struct {
	mutex            sync.Mutex
	config           *common.Config
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

	out := common.MapStr{}
	m.config.Unpack(&out)
	return fmt.Sprintf("config: %v, started=%v, stopped=%v", out.String(), m.started, m.stopped)
}

type mockAdapter struct {
	mutex   sync.Mutex
	configs []*common.Config
	runners []*mockRunner

	CheckConfigCallCount int
}

// CreateConfig generates a valid list of configs from the given event, the received event will have all keys defined by `StartFilter`
func (m *mockAdapter) CreateConfig(event bus.Event) ([]*common.Config, error) {
	if cfgs, ok := event["config"]; ok {
		return cfgs.([]*common.Config), nil
	}
	return m.configs, nil
}

// CheckConfig tests given config to check if it will work or not, returns errors in case it won't work
func (m *mockAdapter) CheckConfig(c *common.Config) error {
	m.CheckConfigCallCount++

	config := struct {
		Broken bool `config:"broken"`
	}{}
	c.Unpack(&config)

	if config.Broken {
		return fmt.Errorf("Broken config")
	}

	return nil
}

func (m *mockAdapter) Create(_ beat.PipelineConnector, config *common.Config) (cfgfile.Runner, error) {
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
	var res []*mockRunner
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
	Registry.AddProvider("mock", func(beatName string, b bus.Bus, uuid uuid.UUID, c *common.Config, k keystore.Keystore) (Provider, error) {
		// intercept bus to mock events
		busChan <- b

		return &mockProvider{}, nil
	})

	// Create a mock adapter
	runnerConfig, _ := common.NewConfigFrom(map[string]string{
		"runner": "1",
	})
	adapter := mockAdapter{
		configs: []*common.Config{runnerConfig},
	}

	// and settings:
	providerConfig, _ := common.NewConfigFrom(map[string]string{
		"type": "mock",
	})
	config := Config{
		Providers: []*common.Config{providerConfig},
	}
	k, _ := keystore.NewFileKeystore("test")
	// Create autodiscover manager
	autodiscover, err := NewAutodiscover("test", nil, &adapter, &adapter, &config, k)
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
		"meta": common.MapStr{
			"foo": "bar",
		},
	})
	wait(t, func() bool { return len(adapter.Runners()) == 1 })

	runners := adapter.Runners()
	assert.Equal(t, len(runners), 1)
	assert.Equal(t, len(autodiscover.configs["mock:foo"]), 1)
	assert.True(t, runners[0].started)
	assert.False(t, runners[0].stopped)

	// Test update
	eventBus.Publish(bus.Event{
		"id":       "foo",
		"provider": "mock",
		"start":    true,
		"meta": common.MapStr{
			"foo": "baz",
		},
	})

	runners = adapter.Runners()
	assert.Equal(t, len(runners), 1)
	assert.Equal(t, len(autodiscover.configs["mock:foo"]), 1)
	assert.True(t, runners[0].started)
	assert.False(t, runners[0].stopped)

	// Test stop/start
	eventBus.Publish(bus.Event{
		"id":       "foo",
		"provider": "mock",
		"stop":     true,
		"meta": common.MapStr{
			"foo": "baz",
		},
	})
	eventBus.Publish(bus.Event{
		"id":       "foo",
		"provider": "mock",
		"start":    true,
		"meta": common.MapStr{
			"foo": "baz",
		},
	})
	wait(t, func() bool { return len(adapter.Runners()) == 2 })

	runners = adapter.Runners()
	assert.Equal(t, len(runners), 2)
	assert.Equal(t, len(autodiscover.configs["mock:foo"]), 1)
	assert.True(t, runners[0].stopped)
	assert.True(t, runners[1].started)
	assert.False(t, runners[1].stopped)

	// Test stop event
	eventBus.Publish(bus.Event{
		"id":       "foo",
		"provider": "mock",
		"stop":     true,
		"meta": common.MapStr{
			"foo": "baz",
		},
	})
	wait(t, func() bool { return adapter.Runners()[1].stopped })

	runners = adapter.Runners()
	assert.Equal(t, len(runners), 2)
	assert.Equal(t, len(autodiscover.configs["mock:foo"]), 0)
	assert.False(t, runners[1].started)
	assert.True(t, runners[1].stopped)
}

func TestAutodiscoverHash(t *testing.T) {
	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	// Register mock autodiscover provider
	busChan := make(chan bus.Bus, 1)

	Registry = NewRegistry()
	Registry.AddProvider("mock", func(beatName string, b bus.Bus, uuid uuid.UUID, c *common.Config, k keystore.Keystore) (Provider, error) {
		// intercept bus to mock events
		busChan <- b

		return &mockProvider{}, nil
	})

	// Create a mock adapter
	runnerConfig1, _ := common.NewConfigFrom(map[string]string{
		"runner": "1",
	})
	runnerConfig2, _ := common.NewConfigFrom(map[string]string{
		"runner": "2",
	})
	adapter := mockAdapter{
		configs: []*common.Config{runnerConfig1, runnerConfig2},
	}

	// and settings:
	providerConfig, _ := common.NewConfigFrom(map[string]string{
		"type": "mock",
	})
	config := Config{
		Providers: []*common.Config{providerConfig},
	}
	k, _ := keystore.NewFileKeystore("test")
	// Create autodiscover manager
	autodiscover, err := NewAutodiscover("test", nil, &adapter, &adapter, &config, k)
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
		"meta": common.MapStr{
			"foo": "bar",
		},
	})
	wait(t, func() bool { return len(adapter.Runners()) == 2 })

	runners := adapter.Runners()
	assert.Equal(t, len(runners), 2)
	assert.Equal(t, len(autodiscover.configs["mock:foo"]), 2)
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
	Registry.AddProvider("mock", func(beatName string, b bus.Bus, uuid uuid.UUID, c *common.Config, k keystore.Keystore) (Provider, error) {
		// intercept bus to mock events
		busChan <- b

		return &mockProvider{}, nil
	})

	// Create a mock adapter that returns a duplicated config
	runnerConfig, _ := common.NewConfigFrom(map[string]string{
		"id": "foo",
	})
	adapter := mockAdapter{
		configs: []*common.Config{runnerConfig, runnerConfig},
	}

	// and settings:
	providerConfig, _ := common.NewConfigFrom(map[string]string{
		"type": "mock",
	})
	config := Config{
		Providers: []*common.Config{providerConfig},
	}
	k, _ := keystore.NewFileKeystore("test")
	// Create autodiscover manager
	autodiscover, err := NewAutodiscover("test", nil, &adapter, &adapter, &config, k)
	if err != nil {
		t.Fatal(err)
	}

	autodiscover.workDone = make(chan struct{})

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
			"meta": common.MapStr{
				"foo": "bar",
			},
		})
		<-autodiscover.workDone
		assert.Equal(t, 1, len(adapter.Runners()), "Only one runner should be started")
		assert.Equal(t, 1, adapter.CheckConfigCallCount, "Check config should have been called only once")
	}
}

func TestAutodiscoverWithConfigCheckFailures(t *testing.T) {
	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	// Register mock autodiscover provider
	busChan := make(chan bus.Bus, 1)
	Registry = NewRegistry()
	Registry.AddProvider("mock", func(beatName string, b bus.Bus, uuid uuid.UUID, c *common.Config, k keystore.Keystore) (Provider, error) {
		// intercept bus to mock events
		busChan <- b

		return &mockProvider{}, nil
	})

	// Create a mock adapter
	runnerConfig1, _ := common.NewConfigFrom(map[string]string{
		"broken": "true",
	})
	runnerConfig2, _ := common.NewConfigFrom(map[string]string{
		"runner": "2",
	})
	adapter := mockAdapter{
		configs: []*common.Config{runnerConfig1, runnerConfig2},
	}

	// and settings:
	providerConfig, _ := common.NewConfigFrom(map[string]string{
		"type": "mock",
	})
	config := Config{
		Providers: []*common.Config{providerConfig},
	}
	k, _ := keystore.NewFileKeystore("test")
	// Create autodiscover manager
	autodiscover, err := NewAutodiscover("test", nil, &adapter, &adapter, &config, k)
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
		"meta": common.MapStr{
			"foo": "bar",
		},
	})

	// As only the second config is valid, total runners will be 1
	wait(t, func() bool { return len(adapter.Runners()) == 1 })
	assert.Equal(t, 1, len(autodiscover.configs["mock:foo"]))
}

func TestAutodiscoverWithMutlipleEntries(t *testing.T) {
	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	// Register mock autodiscover provider
	busChan := make(chan bus.Bus, 1)
	Registry = NewRegistry()
	Registry.AddProvider("mock", func(beatName string, b bus.Bus, uuid uuid.UUID, c *common.Config, k keystore.Keystore) (Provider, error) {
		// intercept bus to mock events
		busChan <- b

		return &mockProvider{}, nil
	})

	// Create a mock adapter
	runnerConfig, _ := common.NewConfigFrom(map[string]string{
		"runner": "1",
	})
	adapter := mockAdapter{
		configs: []*common.Config{runnerConfig},
	}

	// and settings:
	providerConfig, _ := common.NewConfigFrom(map[string]string{
		"type": "mock",
	})
	config := Config{
		Providers: []*common.Config{providerConfig},
	}
	k, _ := keystore.NewFileKeystore("test")
	// Create autodiscover manager
	autodiscover, err := NewAutodiscover("test", nil, &adapter, &adapter, &config, k)
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
		"meta": common.MapStr{
			"foo": "bar",
		},
		"config": []*common.Config{
			common.MustNewConfigFrom(map[string]interface{}{
				"a": "b",
			}),
			common.MustNewConfigFrom(map[string]interface{}{
				"x": "y",
			}),
		},
	})
	wait(t, func() bool { return len(adapter.Runners()) == 2 })

	runners := adapter.Runners()
	assert.Equal(t, len(runners), 2)
	assert.Equal(t, len(autodiscover.configs["mock:foo"]), 2)
	check(t, runners, common.MustNewConfigFrom(map[string]interface{}{"x": "y"}), true, false)
	check(t, runners, common.MustNewConfigFrom(map[string]interface{}{"a": "b"}), true, false)
	// Test start event with changed configurations
	eventBus.Publish(bus.Event{
		"id":       "foo",
		"provider": "mock",
		"start":    true,
		"meta": common.MapStr{
			"foo": "bar",
		},
		"config": []*common.Config{
			common.MustNewConfigFrom(map[string]interface{}{
				"a": "b",
			}),
			common.MustNewConfigFrom(map[string]interface{}{
				"x": "c",
			}),
		},
	})
	wait(t, func() bool { return len(adapter.Runners()) == 3 })
	runners = adapter.Runners()
	// Ensure the first config is the same as before
	fmt.Println(runners)
	assert.Equal(t, len(runners), 3)
	assert.Equal(t, len(autodiscover.configs["mock:foo"]), 2)
	check(t, runners, common.MustNewConfigFrom(map[string]interface{}{"a": "b"}), true, false)

	// Ensure that the runner for the stale config is stopped
	wait(t, func() bool {
		check(t, adapter.Runners(), common.MustNewConfigFrom(map[string]interface{}{"x": "c"}), true, false)
		return true
	})

	// Ensure that the new runner is started
	check(t, runners, common.MustNewConfigFrom(map[string]interface{}{"x": "c"}), true, false)

	// Stop all the configs
	eventBus.Publish(bus.Event{
		"id":       "foo",
		"provider": "mock",
		"stop":     true,
		"meta": common.MapStr{
			"foo": "bar",
		},
		"config": []*common.Config{
			common.MustNewConfigFrom(map[string]interface{}{
				"a": "b",
			}),
			common.MustNewConfigFrom(map[string]interface{}{
				"x": "c",
			}),
		},
	})

	wait(t, func() bool { return adapter.Runners()[2].stopped == true })
	runners = adapter.Runners()
	check(t, runners, common.MustNewConfigFrom(map[string]interface{}{"x": "c"}), false, true)
}

func wait(t *testing.T, test func() bool) {
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

func check(t *testing.T, runners []*mockRunner, expected *common.Config, started, stopped bool) {
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
	out := common.MapStr{}
	expected.Unpack(&out)
	t.Fatalf("expected cfg %v to be started=%v stopped=%v but have %v", out, started, stopped, runners)
}
