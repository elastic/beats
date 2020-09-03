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

package monitorcfg

import (
	"fmt"
	"github.com/elastic/beats/v7/heartbeat/scheduler/schedule"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

// JobConfig represents fields needed to execute a single job.
type JobConfig struct {
	Name     string             `config:"pluginName"`
	Type     string             `config:"type"`
	Schedule *schedule.Schedule `config:"schedule" validate:"required"`
}

// AgentInput represents the base config used when using the agent.
// We expect there to be exactly one stream here, and for that config,
// to map to a single 'regular' config.
type AgentInput struct {
	Id      string           `config:"id"`
	Name    string           `config:"name"`
	Meta    *AgentMeta   	 `config:"meta"`
	DataStream *DataStream `config:"data_stream"`
	Streams []*common.Config `config:"streams" validate:"required"`
	// A regular heartbeat config, suitable for passing to monitor plugin constructor
	// This is set by the UnpackAgentInput constructor
	StandardConfig *common.Config
}

type DataStream struct {
	Namespace string `config:"namespace"`
	Dataset string `config:"dataset"`
	Type string `config:"type"`
}

func (ds *DataStream) IndexName() string {
	return fmt.Sprintf("%s-%s-%s", ds.Type, ds.Dataset, ds.Namespace)
}

type AgentMeta struct {
	Pkg *AgentPackage `config:"package"'`
}

type AgentPackage struct {
	Name string `config:"name"`
	Version string `config:"version"`
}

func UnpackAgentInput(config *common.Config) (ai AgentInput, err error) {
	err = config.Unpack(&ai)
	if err != nil {
		return AgentInput{}, err
	}

	if len(ai.Streams) != 1 {
		return AgentInput{}, fmt.Errorf("received agent config with len(streams)==%d", len(ai.Streams))
	}
	stdConfig := ai.Streams[0]

	// Unpack the single stream's DataStream over the outer DataStream to yield one DataStream with
	// dataset, type and namespace
	dsCfg, err := stdConfig.Child("data_stream", -1)
	if err != nil {
		return AgentInput{}, fmt.Errorf("could access child data_stream: %w", err)
	}
	ds := ai.DataStream
	err = dsCfg.Unpack(&ds)
	if err != nil {
		logp.Warn("AAAH ERROR2")
		return AgentInput{}, fmt.Errorf("could not unpack child data_stream: %w", err)
	}

	// We overwrite the ID of monitor with the input ID since this comes
	// centrally from Kibana and should have greater precedence due to it
	// being part of a persistent store in ES that better tracks the life
	// of a config object than a text file
	if ai.Id != "" {
		err := stdConfig.Merge(common.MapStr{"id": ai.Id, "index": ds.IndexName()})
		if err != nil {
			return AgentInput{}, fmt.Errorf("could not override stream ID with agent ID: %w", err)
		}
	}

	ai.StandardConfig = stdConfig

	return
}
