package prospector

import (
	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/registrar"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
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

// Create creates a prospector based on a config
func (r *RunnerFactory) Create(c *common.Config) (cfgfile.Runner, error) {
	p, err := New(c, r.outlet, r.beatDone, r.registrar.GetStates())
	if err != nil {
		logp.Err("Error creating prospector: %s", err)
		// In case of error with loading state, prospector is still returned
		return p, err
	}

	return p, nil
}
