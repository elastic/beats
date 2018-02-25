package autodiscover

import (
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"

	"github.com/stretchr/testify/assert"
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
func (m *mockAdapter) CheckConfig(*common.Config) error {
	return nil
}

func (m *mockAdapter) Create(config *common.Config, meta *common.MapStrPointer) (cfgfile.Runner, error) {
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
	// Register mock autodiscover provider
	busChan := make(chan bus.Bus, 1)
	Registry = NewRegistry()
	Registry.AddProvider("mock", func(b bus.Bus, c *common.Config) (Provider, error) {
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
	autodiscover, err := NewAutodiscover("test", &adapter, &config)
	if err != nil {
		t.Fatal(err)
	}

	// Start it
	autodiscover.Start()
	defer autodiscover.Stop()
	eventBus := <-busChan

	// Test start event
	eventBus.Publish(bus.Event{
		"start": true,
		"meta": common.MapStr{
			"foo": "bar",
		},
	})
	time.Sleep(10 * time.Millisecond)
	runners := adapter.Runners()
	assert.Equal(t, len(runners), 1)
	assert.Equal(t, runners[0].meta.Get()["foo"], "bar")
	assert.True(t, runners[0].started)
	assert.False(t, runners[0].stopped)

	// Test update
	eventBus.Publish(bus.Event{
		"start": true,
		"meta": common.MapStr{
			"foo": "baz",
		},
	})
	time.Sleep(10 * time.Millisecond)
	runners = adapter.Runners()
	assert.Equal(t, len(runners), 1)
	assert.Equal(t, runners[0].meta.Get()["foo"], "baz") // meta is updated
	assert.True(t, runners[0].started)
	assert.False(t, runners[0].stopped)

	// Test stop/start
	eventBus.Publish(bus.Event{
		"stop": true,
		"meta": common.MapStr{
			"foo": "baz",
		},
	})
	eventBus.Publish(bus.Event{
		"start": true,
		"meta": common.MapStr{
			"foo": "baz",
		},
	})
	time.Sleep(10 * time.Millisecond)
	runners = adapter.Runners()
	assert.Equal(t, len(runners), 2)
	assert.True(t, runners[0].stopped)
	assert.Equal(t, runners[1].meta.Get()["foo"], "baz")
	assert.True(t, runners[1].started)
	assert.False(t, runners[1].stopped)

	// Test stop event
	eventBus.Publish(bus.Event{
		"stop": true,
		"meta": common.MapStr{
			"foo": "baz",
		},
	})
	time.Sleep(10 * time.Millisecond)
	runners = adapter.Runners()
	assert.Equal(t, len(runners), 2)
	assert.Equal(t, runners[1].meta.Get()["foo"], "baz")
	assert.True(t, runners[1].started)
	assert.True(t, runners[1].stopped)
}

func TestAutodiscoverHash(t *testing.T) {
	// Register mock autodiscover provider
	busChan := make(chan bus.Bus, 1)

	Registry = NewRegistry()
	Registry.AddProvider("mock", func(b bus.Bus, c *common.Config) (Provider, error) {
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
	autodiscover, err := NewAutodiscover("test", &adapter, &config)
	if err != nil {
		t.Fatal(err)
	}

	// Start it
	autodiscover.Start()
	defer autodiscover.Stop()
	eventBus := <-busChan

	// Test start event
	eventBus.Publish(bus.Event{
		"start": true,
		"meta": common.MapStr{
			"foo": "bar",
		},
	})
	time.Sleep(10 * time.Millisecond)
	runners := adapter.Runners()
	assert.Equal(t, len(runners), 2)
	assert.Equal(t, runners[0].meta.Get()["foo"], "bar")
	assert.True(t, runners[0].started)
	assert.False(t, runners[0].stopped)
	assert.Equal(t, runners[1].meta.Get()["foo"], "bar")
	assert.True(t, runners[1].started)
	assert.False(t, runners[1].stopped)
}
