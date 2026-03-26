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

// Package addagentmetadata provides a single processor that injects all
// Elastic Agent metadata fields into a beat.Event. It replaces the chain of
// individual add_fields processors that the agent normally prepends to every
// input, avoiding per-event map cloning and deep-update overhead.
package addagentmetadata

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type Config struct {
	InputID      string              `config:"input_id"`
	StreamID     string              `config:"stream_id"`
	DataStream   *DataStreamConfig   `config:"data_stream"`
	ElasticAgent *ElasticAgentConfig `config:"elastic_agent"`
}

type DataStreamConfig struct {
	Dataset   string `config:"dataset"`
	Namespace string `config:"namespace"`
	Type      string `config:"type"`
}
type ElasticAgentConfig struct {
	ID       string `config:"id"`
	Snapshot bool   `config:"snapshot"`
	Version  string `config:"version"`
}

type addAgentMetadata struct {
	cfg Config
}

func New(cfg Config) beat.Processor {
	return &addAgentMetadata{cfg: cfg}
}

func CreateAddAgentMetadata(c *conf.C, _ *logp.Logger) (beat.Processor, error) {
	var cfg Config
	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack add_agent_metadata config: %w", err)
	}
	return New(cfg), nil
}

func (p *addAgentMetadata) Run(event *beat.Event) (*beat.Event, error) {
	if event == nil {
		return nil, nil
	}

	updateMap := make(mapstr.M)
	if p.cfg.DataStream != nil {
		updateMap["data_stream"] = mapstr.M{
			"dataset":   p.cfg.DataStream.Dataset,
			"namespace": p.cfg.DataStream.Namespace,
			"type":      p.cfg.DataStream.Type,
		}
		updateMap["event"] = mapstr.M{
			"dataset": p.cfg.DataStream.Dataset,
		}
	}
	if p.cfg.ElasticAgent != nil {
		updateMap["elastic_agent"] = mapstr.M{
			"id":       p.cfg.ElasticAgent.ID,
			"snapshot": p.cfg.ElasticAgent.Snapshot,
			"version":  p.cfg.ElasticAgent.Version,
		}
		updateMap["agent"] = mapstr.M{
			"id": p.cfg.ElasticAgent.ID, // mirrors elastic_agent.id for convenience
		}
	}

	inputStreamMap := make(mapstr.M)
	if p.cfg.InputID != "" {
		inputStreamMap["input_id"] = p.cfg.InputID
	}
	if p.cfg.StreamID != "" {
		inputStreamMap["stream_id"] = p.cfg.StreamID
	}

	// add input_id and stream_id only if either of them are set
	if len(inputStreamMap) > 0 {
		updateMap["@metadata"] = inputStreamMap
	}

	// insert the metadata and update the event
	event.DeepUpdate(updateMap)
	return event, nil
}

func (p *addAgentMetadata) String() string {
	return fmt.Sprintf("add_agent_metadata=[input_id=%s, elastic_agent.id=%s, data_stream.dataset=%s]",
		p.cfg.InputID, p.cfg.ElasticAgent.ID, p.cfg.DataStream.Dataset)
}
