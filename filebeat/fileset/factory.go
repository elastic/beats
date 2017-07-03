package fileset

import (
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
	"github.com/mitchellh/hashstructure"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/prospector"
	"github.com/elastic/beats/filebeat/registrar"
)

// Factory for modules
type Factory struct {
	outlet         channel.Outleter
	registrar      *registrar.Registrar
	beatVersion    string
	pipelineLoader PipelineLoader
	beatDone       chan struct{}
}

// Wrap an array of prospectors and implements cfgfile.Runner interface
type prospectorsRunner struct {
	id             uint64
	moduleRegistry *ModuleRegistry
	prospectors    []*prospector.Prospector
	pipelineLoader PipelineLoader
}

// NewFactory instantiates a new Factory
func NewFactory(outlet channel.Outleter, registrar *registrar.Registrar, beatVersion string, pipelineLoader PipelineLoader, beatDone chan struct{}) *Factory {
	return &Factory{
		outlet:         outlet,
		registrar:      registrar,
		beatVersion:    beatVersion,
		beatDone:       beatDone,
		pipelineLoader: pipelineLoader,
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
		id:             id,
		moduleRegistry: m,
		prospectors:    prospectors,
		pipelineLoader: f.pipelineLoader,
	}, nil
}

func (p *prospectorsRunner) Start() {
	// Load pipelines
	if p.pipelineLoader != nil {
		// Setup a callback & load now too, as we are already connected
		callback := func(esClient *elasticsearch.Client) error {
			return p.moduleRegistry.LoadPipelines(p.pipelineLoader)
		}
		elasticsearch.RegisterConnectCallback(callback)

		err := p.moduleRegistry.LoadPipelines(p.pipelineLoader)
		if err != nil {
			// Log error and continue
			logp.Err("Error loading pipeline: %s", err)
		}
	}

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
