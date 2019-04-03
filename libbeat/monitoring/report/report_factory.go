package report

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
)

// Factory is factory producing reporters
type Factory struct {
	beat     beat.Info
	outputs  common.ConfigNamespace
	settings Settings
}

// NewFactory returns a factory for creating instances of
func NewFactory(beat beat.Info, outputs common.ConfigNamespace, settings Settings) *Factory {
	return &Factory{
		beat:     beat,
		outputs:  outputs,
		settings: settings,
	}
}

// Create creates a reporter based on a config
func (f *Factory) Create(p beat.Pipeline, c *common.Config, meta *common.MapStrPointer) (cfgfile.Runner, error) {
	reporter, err := New(f.beat, f.settings, c, f.outputs)
	return reporter, err
}

// CheckConfig checks if a config is valid or not
func (f *Factory) CheckConfig(config *common.Config) error {
	// TODO: add code here once we know that spinning up a filebeat input to check for errors doesn't cause memory leaks.
	return nil
}
