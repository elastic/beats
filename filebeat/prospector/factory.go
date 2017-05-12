package prospector

import (
	"github.com/elastic/beats/filebeat/registrar"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type Factory struct {
	outlet    Outlet
	registrar *registrar.Registrar
	beatDone  chan struct{}
}

func NewFactory(outlet Outlet, registrar *registrar.Registrar, beatDone chan struct{}) *Factory {
	return &Factory{
		outlet:    outlet,
		registrar: registrar,
		beatDone:  beatDone,
	}
}

func (r *Factory) Create(c *common.Config) (cfgfile.Runner, error) {

	p, err := NewProspector(c, r.outlet, r.beatDone)
	if err != nil {
		logp.Err("Error creating prospector: %s", err)
		return nil, err
	}

	err = p.LoadStates(r.registrar.GetStates())
	if err != nil {
		logp.Err("Error loading states for prospector %v: %v", p.ID(), err)
		// In case of error with loading state, prospector is still returne
		return p, err
	}

	return p, nil
}
