package fileset

import (
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/mitchellh/hashstructure"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/prospector"
	"github.com/elastic/beats/filebeat/registrar"
)

// Factory is a factory for registrars
type Factory struct {
	outlet      channel.Outleter
	registrar   *registrar.Registrar
	beatVersion string
	beatDone    chan struct{}
}

// Wrap an array of prospectors and implements cfgfile.Runner interface
type prospectorsRunner struct {
	id          uint64
	prospectors []*prospector.Prospector
}

// NewFactory instantiates a new Factory
func NewFactory(outlet channel.Outleter, registrar *registrar.Registrar, beatVersion string, beatDone chan struct{}) *Factory {
	return &Factory{
		outlet:      outlet,
		registrar:   registrar,
		beatVersion: beatVersion,
		beatDone:    beatDone,
	}
}

// Create creates a module based on a config
func (f *Factory) Create(c *common.Config) (cfgfile.Runner, error) {
	// Start a registry of one module:
	m, err := NewModuleRegistry([]*common.Config{c}, f.beatVersion)
	if err != nil {
		return nil, err
	}

	pConfigs, err := m.GetProspectorConfigs()
	if err != nil {
		return nil, err
	}

	// Hash module ID
	var h map[string]interface{}
	c.Unpack(&h)
	id, err := hashstructure.Hash(h, nil)
	if err != nil {
		return nil, err
	}

	prospectors := make([]*prospector.Prospector, len(pConfigs))
	for i, pConfig := range pConfigs {
		prospectors[i], err = prospector.NewProspector(pConfig, f.outlet, f.beatDone, f.registrar.GetStates())
		if err != nil {
			logp.Err("Error creating prospector: %s", err)
			return nil, err
		}
	}

	return &prospectorsRunner{
		id:          id,
		prospectors: prospectors,
	}, nil
}

func (p *prospectorsRunner) Start() {
	for _, prospector := range p.prospectors {
		prospector.Start()
	}
}
func (p *prospectorsRunner) Stop() {
	for _, prospector := range p.prospectors {
		prospector.Stop()
	}
}
func (p *prospectorsRunner) ID() uint64 {
	return p.id
}
