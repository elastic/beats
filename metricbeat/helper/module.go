package helper

import (
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/urso/ucfg"
)

// Module specifics. This must be defined by each module
// The module object has to configs: BaseConfig and RawConfig. BaseConfig only containts the common fields
// across all modules. RawConfig contains the unprocessed configuration which must be processed by the
// specific implementation of the module on Setup
type Module struct {
	Name    string
	Enabled bool

	// Moduler implementation
	Moduler Moduler

	// List of all metricsets in this module
	MetricSets map[string]*MetricSet

	Config *ucfg.Config

	// MetricSet waitgroup
	wg sync.WaitGroup
}

// NewModule creates a new module
func NewModule(name string, moduler Moduler) *Module {
	return &Module{
		Name:       name,
		Moduler:    moduler,
		Enabled:    false,
		MetricSets: map[string]*MetricSet{},
		wg:         sync.WaitGroup{},
	}
}

// Registers moudle with central registry
func (m *Module) Register() {
	Registry.AddModule(m)
}

// Add metric to module
func (m *Module) AddMetric(metricSet *MetricSet) {
	m.MetricSets[metricSet.Name] = metricSet
}

// Interface for each module
type Moduler interface {
	Setup() error
}

// Base configuration for list of modules
type ModulesConfig struct {
	Modules map[string]ModuleConfig
}

// Base module configuration
type ModuleConfig struct {
	Hosts      []string
	Period     string
	MetricSets map[string]MetricSetConfig `yaml:"metricsets"`
}

// Helper functions to easily access default configurations
func (m *Module) GetPeriod() (time.Duration, error) {
	baseConfig := struct{ Period string }{
		Period: "1s",
	}

	if err := m.Config.Unpack(&baseConfig); err != nil {
		return time.Duration(0), err
	}
	return time.ParseDuration(baseConfig.Period)
}

func (m *Module) GetHosts() []string {
	config := struct{ Hosts []string }{}
	err := m.Config.Unpack(&config)
	if err != nil {
		logp.Err("Failed to read hosts: %v", err)
	}
	return config.Hosts
}

// Loads the configurations specific config.
// This needs the configuration object defined inside the module
func (m *Module) LoadConfig(config interface{}) error {
	err := m.Config.Unpack(config)
	if err != nil {
		logp.Err("Load module config error: %v", err)
	}
	return err
}

// Starts the given module
func (m *Module) Start(b *beat.Beat) {

	defer func() {
		if r := recover(); r != nil {
			logp.Err("Module %s paniced and stopped running. Reason: %+v", m.Name, r)
		}
	}()

	if !m.Enabled {
		logp.Debug("helper", "Not starting module %s as not enabled.", m.Name)
		return
	}

	logp.Info("Start Module: %v", m.Name)

	err := m.Moduler.Setup()
	if err != nil {
		logp.Err("Error setting up module: %s. Not starting metricsets for this module.", err)
		return
	}

	for _, metricSet := range m.MetricSets {
		go metricSet.Start(b, m.wg)
		m.wg.Add(1)
	}
}

// Stop stops module and all its metricSets
func (m *Module) Stop() {
	logp.Info("Stopping module: %v", m.Name)
	for _, metricSet := range m.MetricSets {
		go metricSet.Stop()
	}

	m.wg.Wait()
}
