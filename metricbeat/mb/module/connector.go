package module

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/publisher/scheduling"
)

// Connector configures and establishes a beat.Client for publishing events
// to the publisher pipeline.
type Connector struct {
	pipeline           beat.Pipeline
	processors         *processors.Processors
	eventMeta          common.EventMetadata
	dynamicFields      *common.MapStrPointer
	schedulingGroup    string
	schedulingPolicies []beat.SchedulingPolicy
}

type connectorConfig struct {
	Processors           processors.PluginConfig `config:"processors"`
	Scheduling           scheduling.LocalConfig  `config:"scheduling"`
	common.EventMetadata `config:",inline"`      // Fields and tags to add to events.
}

func NewConnector(pipeline beat.Pipeline, c *common.Config, dynFields *common.MapStrPointer) (*Connector, error) {
	config := connectorConfig{}
	if err := c.Unpack(&config); err != nil {
		return nil, err
	}

	processors, err := processors.New(config.Processors)
	if err != nil {
		return nil, err
	}

	schedGroup, schedPolicies, err := scheduling.LoadLocal(config.Scheduling)
	if err != nil {
		return nil, err
	}

	return &Connector{
		pipeline:           pipeline,
		processors:         processors,
		eventMeta:          config.EventMetadata,
		dynamicFields:      dynFields,
		schedulingGroup:    schedGroup,
		schedulingPolicies: schedPolicies,
	}, nil
}

func (c *Connector) Connect() (beat.Client, error) {
	return c.pipeline.ConnectWith(beat.ClientConfig{
		SchedulingGroup:    c.schedulingGroup,
		SchedulingPolicies: c.schedulingPolicies,
		EventMetadata:      c.eventMeta,
		Processor:          c.processors,
		DynamicFields:      c.dynamicFields,
	})
}
