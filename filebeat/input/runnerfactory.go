package input

import (
	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/registrar"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
)

// RunnerFactory is a factory for registrars
type RunnerFactory struct {
	outlet    channel.Factory
	registrar *registrar.Registrar
	beatDone  chan struct{}
}

// NewRunnerFactory instantiates a new RunnerFactory
func NewRunnerFactory(outlet channel.Factory, registrar *registrar.Registrar, beatDone chan struct{}) *RunnerFactory {
	return &RunnerFactory{
		outlet:    outlet,
		registrar: registrar,
		beatDone:  beatDone,
	}
}

// Create creates a input based on a config
func (r *RunnerFactory) Create(c *common.Config, meta *common.MapStrPointer) (cfgfile.Runner, error) {
	p, err := New(c, r.outlet, r.beatDone, r.registrar.GetStates(), meta)
	if err != nil {
		// In case of error with loading state, input is still returned
		return p, err
	}

	return p, nil
}
