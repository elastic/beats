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
inputs:
- type: network/flows
  timeout: 10s
  period: 10s
  keep_null: false
  data_stream.namespace: default
- type: network/amqp
  ports: [5672]
  data_stream.namespace: default
`)
	require.NoError(t, err)
	config := Config{}
	require.NoError(t, cfg.Unpack(&config))

	config, err = config.Normalize()
	require.NoError(t, err)

	require.Equal(t, config.Flows.Timeout, "10s")
	require.Len(t, config.ProtocolsList, 1)
}
