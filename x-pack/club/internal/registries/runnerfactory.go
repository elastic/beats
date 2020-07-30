package registries

import (
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/go-concert/unison"
)

// RunnerFactoryRegistry wraps a runner factory and makes it available with the
// filebeat v2 input API. Config validation is best effort and needs to be
// defered for until the input is actually run. We can't tell for sure in advance
// if the input type exists when the plugin is configured.
// Some beats allow some introspection of existing input types, which can be
// exposed to the runnerFactoryRegistry by implementing has.
type RunnerFactoryRegistry struct {
	TypeField string
	Factory   cfgfile.RunnerFactory
	Has       func(string) bool
}

// runnerFactoryInput provides a lazily configured input for a cfgfile.RunnerFactory, that will
// configure and start a runner on Run only.
// Configuration is postponed to the last step, to allow the input to be used
// with limited capabilities of different Beats.
type runnerFactoryInput struct {
	registry *RunnerFactoryRegistry
	config   *common.Config
}

// runnerFactoryPluginManager provides a simply InputManager for use with cfgfile.RunnerFactory.
// The manager ensures that we can configure a Runner that will be usable with the v2 input API.
type runnerFactoryPluginManager struct {
	registry *RunnerFactoryRegistry
	name     string
}

func (r *RunnerFactoryRegistry) Init(_ unison.Group, _ v2.Mode) error { return nil }
func (r *RunnerFactoryRegistry) Find(name string) (v2.Plugin, bool) {
	if r.Has != nil && !r.Has(name) {
		return v2.Plugin{}, false
	}
	return v2.Plugin{
		Name:      name,
		Stability: feature.Stable, // don't generate logs
		Manager:   &runnerFactoryPluginManager{registry: r, name: name},
	}, true
}

func (m *runnerFactoryPluginManager) Init(grp unison.Group, mode v2.Mode) error { return nil }
func (m *runnerFactoryPluginManager) Create(cfg *common.Config) (v2.Input, error) {
	cfg.SetString(m.registry.TypeField, -1, m.name)

	// we might learn that the monitor type does not exist here, although we already have an
	// configured input. But for the scope of 'monitor' as class of inputs this might be okay
	if err := m.registry.Factory.CheckConfig(cfg); err != nil {
		return nil, err
	}

	return &runnerFactoryInput{m.registry, cfg}, nil
}

func (r *runnerFactoryInput) Name() string {
	s, _ := r.config.String(r.registry.TypeField, -1)
	return s
}
func (r *runnerFactoryInput) Test(_ v2.TestContext) error {
	return nil
}
func (r *runnerFactoryInput) Run(ctx v2.Context, pipeline beat.PipelineConnector) error {
	runner, err := r.registry.Factory.Create(pipeline, r.config)
	if err != nil {
		return err
	}

	runner.Start()
	defer runner.Stop()
	<-ctx.Cancelation.Done()
	return nil
}
