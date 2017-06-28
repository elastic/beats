package harvester

import (
	"errors"

	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
)

// Outlet interface is used for forwarding events
type Outlet interface {
	SetSignal(signal <-chan struct{})
	OnEventSignal(data *util.Data) bool
	OnEvent(data *util.Data) bool
}

// Forwarder contains shared options between all harvesters needed to forward events
type Forwarder struct {
	Config     ForwarderConfig
	Outlet     Outlet
	Processors *processors.Processors
}

// ForwarderConfig contains all config options shared by all harvesters
type ForwarderConfig struct {
	common.EventMetadata `config:",inline"`      // Fields and tags to add to events.
	Type                 string                  `config:"type"`
	Pipeline             string                  `config:"pipeline"`
	Module               string                  `config:"_module_name"`  // hidden option to set the module name
	Fileset              string                  `config:"_fileset_name"` // hidden option to set the fileset name
	Processors           processors.PluginConfig `config:"processors"`
}

// NewForwarder creates a new forwarder instances and initialises processors if configured
func NewForwarder(cfg *common.Config, outlet Outlet) (*Forwarder, error) {

	config := ForwarderConfig{}
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	processors, err := processors.New(config.Processors)
	if err != nil {
		return nil, err
	}

	f := &Forwarder{
		Outlet:     outlet,
		Processors: processors,
		Config:     config,
	}

	return f, nil

}

// Send updates the prospector state and sends the event to the spooler
// All state updates done by the prospector itself are synchronous to make sure not states are overwritten
func (f *Forwarder) Send(data *util.Data) error {

	// Add additional prospector meta data to the event
	data.Meta.Pipeline = f.Config.Pipeline
	data.Meta.Module = f.Config.Module
	data.Meta.Fileset = f.Config.Fileset

	if data.HasEvent() {
		data.Event[common.EventMetadataKey] = f.Config.EventMetadata
		data.Event.Put("prospector.type", f.Config.Type)

		// run the filters before sending to spooler
		data.Event = f.Processors.Run(data.Event)
	}

	ok := f.Outlet.OnEventSignal(data)
	if !ok {
		logp.Info("Prospector outlet closed")
		return errors.New("prospector outlet closed")
	}

	return nil
}
