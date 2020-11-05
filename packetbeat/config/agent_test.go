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

package config

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
)

func TestAgentInputNormalization(t *testing.T) {
	cfg, err := common.NewConfigFrom(`
type: packet
data_stream:
  namespace: default
processors:
  - add_fields:
      target: 'elastic_agent'
      fields:
        id: agent-id
        version: 8.0.0
        snapshot: false
streams:
  - type: flow
    timeout: 10s
    period: 10s
    keep_null: false
    data_stream:
      dataset: packet.flow
      type: logs
  - type: icmp
    data_stream:
      dataset: packet.icmp
      type: logs
`)
	require.NoError(t, err)
	config, err := NewAgentConfig(cfg)
	require.NoError(t, err)

	require.Equal(t, config.Flows.Timeout, "10s")
	require.Equal(t, config.Flows.Index, "logs-packet.flow-default")
	require.Len(t, config.ProtocolsList, 1)

	var protocol map[string]interface{}
	require.NoError(t, config.ProtocolsList[0].Unpack(&protocol))
	require.Len(t, protocol["processors"].([]interface{}), 3)
}
