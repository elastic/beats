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

package monitors

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/heartbeat/scheduler"
)

func TestMonitor(t *testing.T) {
	serverMonConf := mockPluginConf(t, "@every 1ms", "http://example.net")
	reg := mockPluginsReg()
	pipelineConnector := &MockPipelineConnector{}

	sched := scheduler.New(1)
	err := sched.Start()
	require.NoError(t, err)
	defer sched.Stop()

	mon, err := newMonitor(serverMonConf, reg, pipelineConnector, sched, false)
	require.NoError(t, err)

	mon.Start()
	defer mon.Stop()

	require.Equal(t, 1, len(pipelineConnector.clients))
	pcClient := pipelineConnector.clients[0]

	time.Sleep(time.Duration(100 * time.Millisecond))

	count := len(pcClient.Publishes())
	assert.Condition(t, func() (success bool) {
		return count >= 1
	}, "expected publishes to be >= 1, got %d", count)

	mon.Stop()
	assert.Equal(t, true, pcClient.closed)
}
