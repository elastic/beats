package prospector

import (
	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/registrar"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// RegistrarContext is a factory for registrars
type RegistrarContext struct {
	outlet    channel.OutleterFactory
	registrar *registrar.Registrar
	beatDone  chan struct{}
}

// NewRegistrarContext instantiates a new RegistrarContext
func NewRegistrarContext(outlet channel.OutleterFactory, registrar *registrar.Registrar, beatDone chan struct{}) *RegistrarContext {
	return &RegistrarContext{
		outlet:    outlet,
		registrar: registrar,
		beatDone:  beatDone,
	}
}

// Create creates a prospector based on a config
func (r *RegistrarContext) Create(c *common.Config) (cfgfile.Runner, error) {
	p, err := NewProspector(c, r.outlet, r.beatDone, r.registrar.GetStates())
	if err != nil {
		logp.Err("Error creating prospector: %s", err)
		// In case of error with loading state, prospector is still returned
		return p, err
	}

	return p, nil
}
