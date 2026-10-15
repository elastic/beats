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

package addagentmetadata

import (
	"encoding/json"
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
	cfg    Config
	fields mapstr.M
	meta   mapstr.M
}

func New(cfg Config) beat.Processor {
	p := &addAgentMetadata{cfg: cfg, fields: mapstr.M{}, meta: mapstr.M{}}

	if cfg.DataStream != nil {
		dsMap := mapstr.M{}
		if cfg.DataStream.Dataset != "" {
			dsMap["dataset"] = cfg.DataStream.Dataset
			p.fields["event"] = mapstr.M{"dataset": cfg.DataStream.Dataset}
		}
		if cfg.DataStream.Namespace != "" {
			dsMap["namespace"] = cfg.DataStream.Namespace
		}
		if cfg.DataStream.Type != "" {
			dsMap["type"] = cfg.DataStream.Type
		}
		if len(dsMap) > 0 {
			p.fields["data_stream"] = dsMap
		}
	}

	if cfg.ElasticAgent != nil {
		elasticAgentMap := mapstr.M{"snapshot": cfg.ElasticAgent.Snapshot}
		if cfg.ElasticAgent.ID != "" {
			elasticAgentMap["id"] = cfg.ElasticAgent.ID
			p.fields["agent"] = mapstr.M{"id": cfg.ElasticAgent.ID}
		}
		if cfg.ElasticAgent.Version != "" {
			elasticAgentMap["version"] = cfg.ElasticAgent.Version
		}
		p.fields["elastic_agent"] = elasticAgentMap
	}

	if cfg.InputID != "" {
		p.meta["input_id"] = cfg.InputID
	}
	if cfg.StreamID != "" {
		p.meta["stream_id"] = cfg.StreamID
	}

	return p
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

	if len(p.fields) > 0 {
		if event.Fields == nil {
			event.Fields = mapstr.M{}
		}
		event.Fields.DeepCloneUpdate(p.fields)
	}

	if len(p.meta) > 0 {
		if event.Meta == nil {
			event.Meta = mapstr.M{}
		}
		event.Meta.DeepCloneUpdate(p.meta)
	}

	return event, nil
}

func (p *addAgentMetadata) String() string {
	s, _ := json.Marshal(p.cfg)
	return fmt.Sprintf("add_agent_metadata=%s", s)
}
