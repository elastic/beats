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

package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	conf "github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/pkg/mapstr"
)

// data_stream is optional so that inputs whose beat routes events itself
// (Heartbeat synthetics) can add the agent metadata without overwriting
// data_stream.* and event.dataset.
func TestAddAgentMetadataWithoutDataStream(t *testing.T) {
	cfg, err := conf.NewConfigFrom(map[string]any{
		"add_agent_metadata": map[string]any{
			"elastic_agent": map[string]any{"id": "agent-id", "version": "9.0.0"},
			"input_id":      "input",
		},
	})
	require.NoError(t, err, "building config")
	p, err := processors.New(processors.PluginConfig{cfg}, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err, "add_agent_metadata without data_stream must be accepted")

	event := &beat.Event{Fields: mapstr.M{"data_stream": mapstr.M{"dataset": "browser.network"}, "event": mapstr.M{"dataset": "browser.network"}}}
	got, err := p.Run(event)
	require.NoError(t, err, "running processor")

	flat := got.Fields.Flatten()
	assert.Equal(t, "browser.network", flat["data_stream.dataset"], "data_stream.dataset must be untouched")
	assert.Equal(t, "browser.network", flat["event.dataset"], "event.dataset must be untouched")
	assert.Equal(t, "agent-id", flat["elastic_agent.id"], "elastic_agent still added")
}

func TestAddAgentMetadataRequiresElasticAgent(t *testing.T) {
	cfg, err := conf.NewConfigFrom(map[string]any{
		"add_agent_metadata": map[string]any{"input_id": "input"},
	})
	require.NoError(t, err, "building config")
	_, err = processors.New(processors.PluginConfig{cfg}, logptest.NewTestingLogger(t, ""))
	assert.Error(t, err, "elastic_agent must stay required")
}
