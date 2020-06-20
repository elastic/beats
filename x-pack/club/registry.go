package main

import (
	"strings"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/go-concert/unison"
)

// registryList combines a list of input registries into a single registry.
// When configuring an input the list tries each registry. The first registry
// that returns an input type wins.
// registry in the list should have a type prefix to allow some routing.
//
// The registryList can be used to combine v2 style inputs and old RunnerFactory
// into a single namespace. By listing v2 style inputs first we can shadow older implementations
// without fully replacing them in the Beats code-base.
type registryList []v2.Registry

func (r registryList) Init(grp unison.Group, mode v2.Mode) error {
	for _, sub := range r {
		if err := sub.Init(grp, mode); err != nil {
			return err
		}
	}
	return nil
}

func (r registryList) Find(name string) (v2.Plugin, bool) {
	for _, sub := range r {
		if p, ok := sub.Find(name); ok {
			return p, true
		}
	}
	return v2.Plugin{}, false
}

// prefix registry routes configuration with a common prefix to the given registry.
// If a configured input type does not has the prefix of the prefixRegistry,
// the registry fails.
type prefixRegistry struct {
	v2.Registry
	prefix string
}

// withTypePrefix wraps a Registry into a prefixRegistry. All inputs in the input registry are now addressable via the common prefix only.
// For example this setup:
//
//    reg = withTypePrefix("logs", filebeatInputs)
//
// requires the input configuration to load the journald input like this:
//
//    - type: logs/journald
//
func withTypePrefix(name string, reg v2.Registry) v2.Registry {
	return &prefixRegistry{Registry: reg, prefix: name + "/"}
}

func (r *prefixRegistry) Find(name string) (v2.Plugin, bool) {
	if !strings.HasPrefix(name, r.prefix) {
		return v2.Plugin{}, false
	}
	return r.Registry.Find(name[len(r.prefix):])
}

// runnerFactoryRegistry wraps a runner factory and makes it available with the
// filebeat v2 input API. Config validation is best effort and needs to be
// defered for until the input is actually run. We can't tell for sure in advance
// if the input type exists when the plugin is configured.
// Some beats allow some introspection of existing input types, which can be
// exposed to the runnerFactoryRegistry by implementing has.
type runnerFactoryRegistry struct {
	typeField string
	factory   cfgfile.RunnerFactory
	has       func(string) bool
}

func (r *runnerFactoryRegistry) Init(_ unison.Group, _ v2.Mode) error { return nil }
func (r *runnerFactoryRegistry) Find(name string) (v2.Plugin, bool) {
	if r.has != nil && !r.has(name) {
		return v2.Plugin{}, false
	}
	return v2.Plugin{
		Name:      name,
		Stability: feature.Stable, // don't generate logs
		Manager:   &runnerFactoryPluginManager{registry: r, name: name},
	}, true
}

// runnerFactoryPluginManager provides a simply InputManager for use with cfgfile.RunnerFactory.
// The manager ensures that we can configure a Runner that will be usable with the v2 input API.
type runnerFactoryPluginManager struct {
	registry *runnerFactoryRegistry
	name     string
}

func (m *runnerFactoryPluginManager) Init(grp unison.Group, mode v2.Mode) error { return nil }
func (m *runnerFactoryPluginManager) Create(cfg *common.Config) (v2.Input, error) {
	cfg.SetString(m.registry.typeField, -1, m.name)

	// we might learn that the monitor type does not exist here, although we already have an
	// configured input. But for the scope of 'monitor' as class of inputs this might be okay
	if err := m.registry.factory.CheckConfig(cfg); err != nil {
		return nil, err
	}

	return &runnerFactoryInput{m.registry, cfg}, nil
}

// runnerFactoryInput provides a lazily configured input for a cfgfile.RunnerFactory, that will
// configure and start a runner on Run only.
// Configuration is postponed to the last step, to allow the input to be used
// with limited capabilities of different Beats.
type runnerFactoryInput struct {
	registry *runnerFactoryRegistry
	config   *common.Config
}

func (r *runnerFactoryInput) Name() string {
	s, _ := r.config.String(r.registry.typeField, -1)
	return s
}
func (r *runnerFactoryInput) Test(_ v2.TestContext) error {
	return nil
}
func (r *runnerFactoryInput) Run(ctx v2.Context, pipeline beat.PipelineConnector) error {
	runner, err := r.registry.factory.Create(pipeline, r.config)
	if err != nil {
		return err
	}

	runner.Start()
	defer runner.Stop()
	<-ctx.Cancelation.Done()
	return nil
}
