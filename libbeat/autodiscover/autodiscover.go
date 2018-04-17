package autodiscover

import (
	"github.com/elastic/beats/libbeat/autodiscover/meta"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/mitchellh/hashstructure"
)

const debugK = "autodiscover"

// TODO autodiscover providers config reload

// Adapter must be implemented by the beat in order to provide Autodiscover
type Adapter interface {

	// CreateConfig generates a valid list of configs from the given event, the received event will have all keys defined by `StartFilter`
	CreateConfig(bus.Event) ([]*common.Config, error)

	// CheckConfig tests given config to check if it will work or not, returns errors in case it won't work
	CheckConfig(*common.Config) error

	// RunnerFactory provides runner creation by feeding valid configs
	cfgfile.RunnerFactory

	// EventFilter returns the bus filter to retrieve runner start/stop triggering events
	EventFilter() []string
}

// Autodiscover process, it takes a beat adapter and user config and runs autodiscover process, spawning
// new modules when any configured providers does a match
type Autodiscover struct {
	bus       bus.Bus
	adapter   Adapter
	providers []Provider
	runners   *cfgfile.Registry
	meta      *meta.Map

	listener bus.Listener
}

// NewAutodiscover instantiates and returns a new Autodiscover manager
func NewAutodiscover(name string, adapter Adapter, config *Config) (*Autodiscover, error) {
	// Init Event bus
	bus := bus.New(name)

	// Init providers
	var providers []Provider
	for _, providerCfg := range config.Providers {
		provider, err := Registry.BuildProvider(bus, providerCfg)
		if err != nil {
			return nil, err
		}
		logp.Debug(debugK, "Configured autodiscover provider: %s", provider)
		providers = append(providers, provider)
	}

	return &Autodiscover{
		bus:       bus,
		adapter:   adapter,
		runners:   cfgfile.NewRegistry(),
		providers: providers,
		meta:      meta.NewMap(),
	}, nil
}

// Start autodiscover process
func (a *Autodiscover) Start() {
	if a == nil {
		return
	}

	logp.Info("Starting autodiscover manager")
	a.listener = a.bus.Subscribe(a.adapter.EventFilter()...)

	for _, provider := range a.providers {
		provider.Start()
	}

	go a.worker()
}

func (a *Autodiscover) worker() {
	for event := range a.listener.Events() {
		// This will happen on Stop:
		if event == nil {
			return
		}

		if _, ok := event["start"]; ok {
			a.handleStart(event)
		}
		if _, ok := event["stop"]; ok {
			a.handleStop(event)
		}
	}
}

func (a *Autodiscover) handleStart(event bus.Event) {
	configs, err := a.adapter.CreateConfig(event)
	if err != nil {
		logp.Debug(debugK, "Could not generate config from event %v: %v", event, err)
		return
	}
	logp.Debug(debugK, "Got a start event: %v, generated configs: %+v", event, configs)

	meta := getMeta(event)
	for _, config := range configs {
		rawCfg := map[string]interface{}{}
		if err := config.Unpack(rawCfg); err != nil {
			logp.Debug(debugK, "Error unpacking config: %v", err)
			continue
		}

		hash, err := hashstructure.Hash(rawCfg, nil)
		if err != nil {
			logp.Debug(debugK, "Could not hash config %v: %v", config, err)
			continue
		}

		err = a.adapter.CheckConfig(config)
		if err != nil {
			logp.Debug(debugK, "Check failed for config %v: %v, won't start runner", config, err)
			continue
		}

		// Update meta no matter what
		dynFields := a.meta.Store(hash, meta)

		if a.runners.Has(hash) {
			logp.Debug(debugK, "Config %v is already running", config)
			continue
		}

		runner, err := a.adapter.Create(config, &dynFields)
		if err != nil {
			logp.Debug(debugK, "Failed to create runner with config %v: %v", config, err)
			continue
		}

		logp.Info("Autodiscover starting runner: %s", runner)
		a.runners.Add(hash, runner)
		runner.Start()
	}
}

func (a *Autodiscover) handleStop(event bus.Event) {
	configs, err := a.adapter.CreateConfig(event)
	if err != nil {
		logp.Debug(debugK, "Could not generate config from event %v: %v", event, err)
		return
	}
	logp.Debug(debugK, "Got a stop event: %v, generated configs: %+v", event, configs)

	for _, config := range configs {
		rawCfg := map[string]interface{}{}
		if err := config.Unpack(rawCfg); err != nil {
			logp.Debug(debugK, "Error unpacking config: %v", err)
			continue
		}

		hash, err := hashstructure.Hash(rawCfg, nil)
		if err != nil {
			logp.Debug(debugK, "Could not hash config %v: %v", config, err)
			continue
		}

		if !a.runners.Has(hash) {
			logp.Debug(debugK, "Config %v is not running", config)
			continue
		}

		if runner := a.runners.Get(hash); runner != nil {
			logp.Info("Autodiscover stopping runner: %s", runner)
			runner.Stop()
			a.runners.Remove(hash)
		} else {
			logp.Debug(debugK, "Runner not found for stopping: %s", hash)
		}
	}
}

func getMeta(event bus.Event) common.MapStr {
	m := event["meta"]
	if m == nil {
		return nil
	}

	logp.Debug(debugK, "Got a meta field in the event")
	meta, ok := m.(common.MapStr)
	if !ok {
		logp.Err("Got a wrong meta field for event %v", event)
		return nil
	}
	return meta
}

// Stop autodiscover process
func (a *Autodiscover) Stop() {
	if a == nil {
		return
	}

	// Stop listening for events
	a.listener.Stop()

	// Stop providers
	for _, provider := range a.providers {
		provider.Stop()
	}

	// Stop runners
	for hash, runner := range a.runners.CopyList() {
		runner.Stop()
		a.meta.Remove(hash)
	}
	logp.Info("Stopped autodiscover manager")
}
