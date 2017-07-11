package channel

import (
	"sync"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/publisher/bc/publisher"
	"github.com/elastic/beats/libbeat/publisher/beat"
)

type OutletFactory struct {
	done     <-chan struct{}
	pipeline publisher.Publisher

	eventer  beat.ClientEventer
	wgEvents *sync.WaitGroup
}

// clientEventer adjusts wgEvents if events are dropped during shutdown.
type clientEventer struct {
	wgEvents *sync.WaitGroup
}

// prospectorOutletConfig defines common prospector settings
// for the publisher pipline.
type prospectorOutletConfig struct {
	// event processing
	common.EventMetadata `config:",inline"`      // Fields and tags to add to events.
	Processors           processors.PluginConfig `config:"processors"`

	// implicit event fields
	Type string `config:"type"` // prospector.type

	// hidden filebeat modules settings
	Module  string `config:"_module_name"`  // hidden setting
	Fileset string `config:"_fileset_name"` // hidden setting

	// Output meta data settings
	Pipeline string `config:"pipeline"` // ES Ingest pipeline name

}

// NewOutletFactory creates a new outlet factory for
// connecting a prospector to the publisher pipeline.
func NewOutletFactory(
	done <-chan struct{},
	pipeline publisher.Publisher,
	wgEvents *sync.WaitGroup,
) *OutletFactory {
	o := &OutletFactory{
		done:     done,
		pipeline: pipeline,
		wgEvents: wgEvents,
	}

	if wgEvents != nil {
		o.eventer = &clientEventer{wgEvents}
	}

	return o
}

// Create builds a new Outleter, while applying common prospector settings.
// Prospectors and all harvesters use the same pipeline client instance.
// This guarantees ordering between events as required by the registrar for
// file.State updates
func (f *OutletFactory) Create(cfg *common.Config) (Outleter, error) {
	config := prospectorOutletConfig{}
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	processors, err := processors.New(config.Processors)
	if err != nil {
		return nil, err
	}

	setMeta := func(to common.MapStr, key, value string) {
		if value != "" {
			to[key] = value
		}
	}

	meta := common.MapStr{}
	setMeta(meta, "pipeline", config.Pipeline)

	fields := common.MapStr{}
	setMeta(fields, "module", config.Module)
	setMeta(fields, "name", config.Fileset)
	if len(fields) > 0 {
		fields = common.MapStr{
			"fileset": fields,
		}
	}
	if config.Type != "" {
		fields["prospector"] = common.MapStr{
			"type": config.Type,
		}
	}

	client, err := f.pipeline.ConnectX(beat.ClientConfig{
		PublishMode:   beat.GuaranteedSend,
		EventMetadata: config.EventMetadata,
		Meta:          meta,
		Fields:        fields,
		Processor:     processors,
		Events:        f.eventer,
	})
	if err != nil {
		return nil, err
	}

	outlet := newOutlet(client, f.wgEvents)
	if f.done != nil {
		return CloseOnSignal(outlet, f.done), nil
	}
	return outlet, nil
}

func (*clientEventer) Closing()   {}
func (*clientEventer) Closed()    {}
func (*clientEventer) Published() {}

func (c *clientEventer) FilteredOut(_ beat.Event) {}
func (c *clientEventer) DroppedOnPublish(_ beat.Event) {
	c.wgEvents.Done()
}
