package module

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/publisher/beat"
)

// Connector configures ann establishes a beat.Client for publishing events
// to the publisher pipeline.
type Connector struct {
	pipeline   beat.Pipeline
	processors *processors.Processors
	eventMeta  common.EventMetadata
}

type connectorConfig struct {
	Processors           processors.PluginConfig `config:"processors"`
	common.EventMetadata `config:",inline"`      // Fields and tags to add to events.
}

func NewConnector(pipeline beat.Pipeline, c *common.Config) (*Connector, error) {
	config := connectorConfig{}
	if err := c.Unpack(&config); err != nil {
		return nil, err
	}

	processors, err := processors.New(config.Processors)
	if err != nil {
		return nil, err
	}

	return &Connector{
		pipeline:   pipeline,
		processors: processors,
		eventMeta:  config.EventMetadata,
	}, nil
}

func (c *Connector) Connect() (beat.Client, error) {
	return c.pipeline.ConnectWith(beat.ClientConfig{
		EventMetadata: c.eventMeta,
		Processor:     c.processors,
	})
}
