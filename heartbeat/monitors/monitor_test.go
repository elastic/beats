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

	"github.com/elastic/beats/v7/heartbeat/scheduler"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/go-lookslike/testslike"
)

func TestMonitor(t *testing.T) {
	serverMonConf := mockPluginConf(t, "", "@every 1ms", "http://example.net")
	reg, built, closed := mockPluginsReg()
	pipelineConnector := &MockPipelineConnector{}

	sched := scheduler.Create(1, monitoring.NewRegistry(), time.Local, nil, false)
	defer sched.Stop()

	mon, err := newMonitor(serverMonConf, reg, pipelineConnector, sched.Add, nil)
	require.NoError(t, err)

	mon.Start()

	require.Equal(t, 1, len(pipelineConnector.clients))
	pcClient := pipelineConnector.clients[0]

	timeout := time.Second
	start := time.Now()
	success := false
	for time.Since(start) < timeout && !success {
		count := len(pcClient.Publishes())
		if count >= 1 {
			success = true

			pcClient.Close()

			for _, event := range pcClient.Publishes() {
				testslike.Test(t, mockEventMonitorValidator(""), event.Fields)
			}
		} else {
			// Let's yield this goroutine so we don't spin
			// This could (possibly?) lock on a single core system otherwise
			time.Sleep(time.Microsecond)
		}
	}

	if !success {
		t.Fatalf("No publishes detected!")
	}

	assert.Equal(t, 1, built.Load())
	mon.Stop()

	assert.Equal(t, 1, closed.Load())
	assert.Equal(t, true, pcClient.closed)
}

func TestCheckInvalidConfig(t *testing.T) {
	serverMonConf := mockInvalidPluginConf(t)
	reg, built, closed := mockPluginsReg()
	pipelineConnector := &MockPipelineConnector{}

	sched := scheduler.Create(1, monitoring.NewRegistry(), time.Local, nil, false)
	defer sched.Stop()

	m, err := newMonitor(serverMonConf, reg, pipelineConnector, sched.Add, nil)
	require.Error(t, err)
	// This could change if we decide the contract for newMonitor should always return a monitor
	require.Nil(t, m, "For this test to work we need a nil value for the monitor.")

	// These counters are both zero since this fails at config parse time
	require.Equal(t, 0, built.Load())
	require.Equal(t, 0, closed.Load())

	require.Error(t, checkMonitorConfig(serverMonConf, reg))
}
