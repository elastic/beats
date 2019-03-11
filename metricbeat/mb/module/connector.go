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
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
)

// Connector configures and establishes a beat.Client for publishing events
// to the publisher pipeline.
type Connector struct {
	pipeline      beat.Pipeline
	processors    *processors.Processors
	eventMeta     common.EventMetadata
	dynamicFields *common.MapStrPointer
}

type connectorConfig struct {
	Processors           processors.PluginConfig `config:"processors"`
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

	return &Connector{
		pipeline:      pipeline,
		processors:    processors,
		eventMeta:     config.EventMetadata,
		dynamicFields: dynFields,
	}, nil
}

func (c *Connector) Connect() (beat.Client, error) {
	return c.pipeline.ConnectWith(beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			EventMetadata: c.eventMeta,
			Processor:     c.processors,
			DynamicFields: c.dynamicFields,
		},
	})
}
