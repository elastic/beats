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
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/tests/resources"
)

type mockRunner struct {
	mutex            sync.Mutex
	config           *common.Config
	meta             *common.MapStrPointer
	started, stopped bool
}

func (m *mockRunner) Start() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.started = true
}
func (m *mockRunner) Stop() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.stopped = true
}
func (m *mockRunner) Clone() *mockRunner {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return &mockRunner{
		config:  m.config,
		meta:    m.meta,
		started: m.started,
		stopped: m.stopped,
	}
}
func (m *mockRunner) String() string {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return "runner"
}

type mockAdapter struct {
	mutex   sync.Mutex
	configs []*common.Config
	runners []*mockRunner
}

// CreateConfig generates a valid list of configs from the given event, the received event will have all keys defined by `StartFilter`
func (m *mockAdapter) CreateConfig(bus.Event) ([]*common.Config, error) {
	return m.configs, nil
}

// CheckConfig tests given config to check if it will work or not, returns errors in case it won't work
func (m *mockAdapter) CheckConfig(c *common.Config) error {
	config := struct {
		Broken bool `config:"broken"`
	}{}
	c.Unpack(&config)

	if config.Broken {
		fmt.Println("broken")
		return fmt.Errorf("Broken config")
	}

	return nil
}

func (m *mockAdapter) Create(_ beat.Pipeline, config *common.Config, meta *common.MapStrPointer) (cfgfile.Runner, error) {
	runner := &mockRunner{
		config: config,
		meta:   meta,
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
	Registry.AddProvider("mock", func(b bus.Bus, uuid uuid.UUID, c *common.Config) (Provider, error) {
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

	// Create autodiscover manager
	autodiscover, err := NewAutodiscover("test", nil, &adapter, &config)
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
	assert.Equal(t, runners[0].meta.Get()["foo"], "bar")
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
	wait(t, func() bool { return adapter.Runners()[0].meta.Get()["foo"] == "baz" })

	runners = adapter.Runners()
	assert.Equal(t, len(runners), 1)
	assert.Equal(t, len(autodiscover.configs["mock:foo"]), 1)
	assert.Equal(t, runners[0].meta.Get()["foo"], "baz") // meta is updated
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
	assert.Equal(t, runners[1].meta.Get()["foo"], "baz")
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
	assert.Equal(t, runners[1].meta.Get()["foo"], "baz")
	assert.True(t, runners[1].started)
	assert.True(t, runners[1].stopped)
}

func TestAutodiscoverHash(t *testing.T) {
	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	// Register mock autodiscover provider
	busChan := make(chan bus.Bus, 1)

	Registry = NewRegistry()
	Registry.AddProvider("mock", func(b bus.Bus, uuid uuid.UUID, c *common.Config) (Provider, error) {
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

	// Create autodiscover manager
	autodiscover, err := NewAutodiscover("test", nil, &adapter, &config)
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
	assert.Equal(t, runners[0].meta.Get()["foo"], "bar")
	assert.True(t, runners[0].started)
	assert.False(t, runners[0].stopped)
	assert.Equal(t, runners[1].meta.Get()["foo"], "bar")
	assert.True(t, runners[1].started)
	assert.False(t, runners[1].stopped)
}

func TestAutodiscoverWithConfigCheckFailures(t *testing.T) {
	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	// Register mock autodiscover provider
	busChan := make(chan bus.Bus, 1)
	Registry = NewRegistry()
	Registry.AddProvider("mock", func(b bus.Bus, uuid uuid.UUID, c *common.Config) (Provider, error) {
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

	// Create autodiscover manager
	autodiscover, err := NewAutodiscover("test", nil, &adapter, &config)
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
