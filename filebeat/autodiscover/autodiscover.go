package autodiscover

import (
	"errors"

	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
)

// AutodiscoverAdapter for Filebeat modules & input
type AutodiscoverAdapter struct {
	inputFactory  cfgfile.RunnerFactory
	moduleFactory cfgfile.RunnerFactory
}

// NewAutodiscoverAdapter builds and returns an autodiscover adapter for Filebeat modules & input
func NewAutodiscoverAdapter(inputFactory, moduleFactory cfgfile.RunnerFactory) *AutodiscoverAdapter {
	return &AutodiscoverAdapter{
		inputFactory:  inputFactory,
		moduleFactory: moduleFactory,
	}
}

// CreateConfig generates a valid list of configs from the given event, the received event will have all keys defined by `StartFilter`
func (m *AutodiscoverAdapter) CreateConfig(e bus.Event) ([]*common.Config, error) {
	config, ok := e["config"].([]*common.Config)
	if !ok {
		return nil, errors.New("Got a wrong value in event `config` key")
	}
	return config, nil
}

// CheckConfig tests given config to check if it will work or not, returns errors in case it won't work
func (m *AutodiscoverAdapter) CheckConfig(c *common.Config) error {
	// TODO implment config check for all modules
	return nil
}

// Create a module or input from the given config
func (m *AutodiscoverAdapter) Create(c *common.Config, meta *common.MapStrPointer) (cfgfile.Runner, error) {
	if c.HasField("module") {
		return m.moduleFactory.Create(c, meta)
	}
	return m.inputFactory.Create(c, meta)
}

// EventFilter returns the bus filter to retrieve runner start/stop triggering events
func (m *AutodiscoverAdapter) EventFilter() []string {
	return []string{"config"}
}
