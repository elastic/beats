// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package module

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/common/fmtstr"
	"github.com/elastic/beats/v8/libbeat/processors"
	"github.com/elastic/beats/v8/libbeat/processors/add_formatted_index"
)

// Connector configures and establishes a beat.Client for publishing events
// to the publisher pipeline.
type Connector struct {
	pipeline   beat.PipelineConnector
	processors *processors.Processors
	eventMeta  common.EventMetadata
	timeSeries bool
	keepNull   bool
}

type connectorConfig struct {
	Processors processors.PluginConfig `config:"processors"`
	// ES output index pattern
	Index fmtstr.EventFormatString `config:"index"`

	// KeepNull determines whether published events will keep null values or omit them.
	KeepNull bool `config:"keep_null"`

	common.EventMetadata `config:",inline"` // Fields and tags to add to events.
}

type metricSetRegister interface {
	ProcessorsForMetricSet(moduleName, metricSetName string) (*processors.Processors, error)
}

func NewConnector(
	beatInfo beat.Info,
	pipeline beat.PipelineConnector,
	c *common.Config,
) (*Connector, error) {
	config := connectorConfig{}
	if err := c.Unpack(&config); err != nil {
		return nil, err
	}

	processors, err := processorsForConfig(beatInfo, config)
	if err != nil {
		return nil, err
	}

	return &Connector{
		pipeline:   pipeline,
		processors: processors,
		eventMeta:  config.EventMetadata,
		keepNull:   config.KeepNull,
	}, nil
}

// UseMetricSetProcessors appends processors defined in metricset configuration to the connector properties.
func (c *Connector) UseMetricSetProcessors(r metricSetRegister, moduleName, metricSetName string) error {
	metricSetProcessors, err := r.ProcessorsForMetricSet(moduleName, metricSetName)
	if err != nil {
		return errors.Wrapf(err, "reading metricset processors failed (module: %s, metricset: %s)",
			moduleName, metricSetName)
	}

	if metricSetProcessors == nil || len(metricSetProcessors.List) == 0 {
		return nil // no processors are defined
	}

	procs := processors.NewList(nil)
	procs.AddProcessors(*metricSetProcessors)
	for _, p := range c.processors.List {
		procs.AddProcessor(p)
	}
	c.processors = procs
	return nil
}

func (c *Connector) Connect() (beat.Client, error) {
	return c.pipeline.ConnectWith(beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			EventMetadata: c.eventMeta,
			Processor:     c.processors,
			KeepNull:      c.keepNull,
		},
	})
}

// processorsForConfig assembles the Processors for a Connector.
func processorsForConfig(
	beatInfo beat.Info, config connectorConfig,
) (*processors.Processors, error) {
	procs := processors.NewList(nil)

	// Processor order is important! The index processor, if present, must be
	// added before the user processors.
	if !config.Index.IsEmpty() {
		staticFields := fmtstr.FieldsForBeat(beatInfo.Beat, beatInfo.Version)
		timestampFormat, err :=
			fmtstr.NewTimestampFormatString(&config.Index, staticFields)
		if err != nil {
			return nil, err
		}
		indexProcessor := add_formatted_index.New(timestampFormat)
		procs.AddProcessor(indexProcessor)
	}

	userProcs, err := processors.New(config.Processors)
	if err != nil {
		return nil, err
	}
	procs.AddProcessors(*userProcs)

	return procs, nil
}
