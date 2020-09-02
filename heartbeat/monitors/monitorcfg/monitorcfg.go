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
	Meta    *AgentMeta   `config:"meta"`
	Streams []*common.Config `config:"streams" validate:"required"`
}

type AgentMeta struct {
	Pkg *AgentPackage `config:package`
}

type AgentPackage struct {
	Name string `config:name`
	Version string `config:version`
}

// ToStandardConfig transforms this AgentInput into something compatible with
// a JobConfig and whatever more specific monitor type it becomes later.
func (ai AgentInput) ToStandardConfig() (*common.Config, error) {
	if len(ai.Streams) != 1 {
		return nil, fmt.Errorf("received agent config with len(streams)==%d", len(ai.Streams))
	}
	config := ai.Streams[0]

	// We overwrite the ID of monitor with the input ID since this comes
	// centrally from Kibana and should have greater precedence due to it
	// being part of a persistent store in ES that better tracks the life
	// of a config object than a text file
	if ai.Id != "" {
		err := config.SetString("id", 0, ai.Id)
		if err != nil {
			return nil, fmt.Errorf("could not override stream ID with agent ID: %w", err)
		}
	}

	return config, nil
}
