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

package channel

import (
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/fmtstr"
	"github.com/elastic/beats/libbeat/processors"
)

// ConnectorFunc is an adapter for using ordinary functions as Connector.
type ConnectorFunc func(*common.Config, beat.ClientConfig) (Outleter, error)

type pipelineConnector struct {
	parent   *OutletFactory
	pipeline beat.Pipeline
}

// addFormattedIndex is a Processor to set an event's "raw_index" metadata field
// with a given TimestampFormatString. The elasticsearch output interprets
// that field as specifying the (raw string) index the event should be sent to;
// in other outputs it is just included in the metadata.
type addFormattedIndex struct {
	formatString *fmtstr.TimestampFormatString
}

// Connect passes the cfg and the zero value of beat.ClientConfig to the underlying function.
func (fn ConnectorFunc) Connect(cfg *common.Config) (Outleter, error) {
	return fn(cfg, beat.ClientConfig{})
}

// ConnectWith passes the configuration and the pipeline connection setting to the underlying function.
func (fn ConnectorFunc) ConnectWith(cfg *common.Config, clientCfg beat.ClientConfig) (Outleter, error) {
	return fn(cfg, clientCfg)
}

func (c *pipelineConnector) Connect(cfg *common.Config) (Outleter, error) {
	return c.ConnectWith(cfg, beat.ClientConfig{})
}

func (c *pipelineConnector) ConnectWith(cfg *common.Config, clientCfg beat.ClientConfig) (Outleter, error) {
	config := inputOutletConfig{}
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	procs, err := processorsForConfig(c.parent.beatInfo, config, clientCfg)
	if err != nil {
		return nil, err
	}

	setOptional := func(to common.MapStr, key string, value string) {
		if value != "" {
			to.Put(key, value)
		}
	}

	meta := clientCfg.Processing.Meta.Clone()
	fields := clientCfg.Processing.Fields.Clone()

	serviceType := config.ServiceType
	if serviceType == "" {
		serviceType = config.Module
	}

	setOptional(meta, "pipeline", config.Pipeline)
	setOptional(fields, "fileset.name", config.Fileset)
	setOptional(fields, "service.type", serviceType)
	setOptional(fields, "input.type", config.Type)
	if config.Module != "" {
		event := common.MapStr{"module": config.Module}
		if config.Fileset != "" {
			event["dataset"] = config.Module + "." + config.Fileset
		}
		fields["event"] = event
	}

	mode := clientCfg.PublishMode
	if mode == beat.DefaultGuarantees {
		mode = beat.GuaranteedSend
	}

	// connect with updated configuration
	clientCfg.PublishMode = mode
	clientCfg.Processing.EventMetadata = config.EventMetadata
	clientCfg.Processing.Meta = meta
	clientCfg.Processing.Fields = fields
	clientCfg.Processing.Processor = procs
	clientCfg.Processing.KeepNull = config.KeepNull
	client, err := c.pipeline.ConnectWith(clientCfg)
	if err != nil {
		return nil, err
	}

	outlet := newOutlet(client, c.parent.wgEvents)
	if c.parent.done != nil {
		return CloseOnSignal(outlet, c.parent.done), nil
	}
	return outlet, nil
}

// processorsForConfig assembles the Processors for a pipelineConnector.
func processorsForConfig(
	beatInfo beat.Info, config inputOutletConfig, clientCfg beat.ClientConfig,
) (*processors.Processors, error) {
	procs := processors.NewList(nil)

	// Processor ordering is important:
	// 1. Index configuration
	if !config.Index.IsEmpty() {
		staticFields := fmtstr.FieldsForBeat(beatInfo.Beat, beatInfo.Version)
		timestampFormat, err :=
			fmtstr.NewTimestampFormatString(&config.Index, staticFields)
		if err != nil {
			return nil, err
		}
		indexProcessor := &addFormattedIndex{timestampFormat}
		procs.List = append(procs.List, indexProcessor)
	}

	// 2. ClientConfig processors
	if lst := clientCfg.Processing.Processor; lst != nil {
		procs.List = append(procs.List, lst)
	}

	// 3. User processors
	userProcessors, err := processors.New(config.Processors)
	if err != nil {
		return nil, err
	}
	// Subtlety: it is important here that we append the individual elements of
	// userProcessors, rather than userProcessors itself, even though
	// userProcessors implements the processors.Processor interface. This is
	// because the contents of what we return are later pulled out into a
	// processing.group rather than a processors.Processors, and the two have
	// different error semantics: processors.Processors aborts processing on
	// any error, whereas processing.group only aborts on fatal errors. The
	// latter is the most common behavior, and the one we are preserving here for
	// backwards compatibility.
	// We are unhappy about this and have plans to fix this inconsistency at a
	// higher level, but for now we need to respect the existing semantics.
	procs.List = append(procs.List, userProcessors.List...)
	return procs, nil
}

func (p *addFormattedIndex) Run(event *beat.Event) (*beat.Event, error) {
	index, err := p.formatString.Run(event.Timestamp)
	if err != nil {
		return nil, err
	}

	if event.Meta == nil {
		event.Meta = common.MapStr{}
	}
	event.Meta["raw_index"] = index
	return event, nil
}

func (p *addFormattedIndex) String() string {
	return fmt.Sprintf("add_index_pattern=%v", p.formatString)
}
