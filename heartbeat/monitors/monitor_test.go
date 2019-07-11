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

	"github.com/elastic/go-lookslike/testslike"

	"github.com/elastic/beats/heartbeat/scheduler"
)

func TestMonitor(t *testing.T) {
	serverMonConf := mockPluginConf(t, "", "@every 1ms", "http://example.net")
	reg := mockPluginsReg()
	pipelineConnector := &MockPipelineConnector{}

	sched := scheduler.New(1)
	err := sched.Start()
	require.NoError(t, err)
	defer sched.Stop()

	mon, err := newMonitor(serverMonConf, reg, pipelineConnector, sched, false, nil)
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

			mon.Stop()
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

	mon.Stop()
	assert.Equal(t, true, pcClient.closed)
}

func TestDuplicateMonitorIDs(t *testing.T) {
	serverMonConf := mockPluginConf(t, "custom", "@every 1ms", "http://example.net")
	badConf := mockBadPluginConf(t, "custom", "@every 1ms")
	reg := mockPluginsReg()
	pipelineConnector := &MockPipelineConnector{}

	sched := scheduler.New(1)
	err := sched.Start()
	require.NoError(t, err)
	defer sched.Stop()

	makeTestMon := func() (*Monitor, error) {
		return newMonitor(serverMonConf, reg, pipelineConnector, sched, false, nil)
	}

	// Ensure that an error is returned on a bad config
	_, m0Err := newMonitor(badConf, reg, pipelineConnector, sched, false, nil)
	require.Error(t, m0Err)

	// Would fail if the previous newMonitor didn't free the monitor.id
	m1, m1Err := makeTestMon()
	require.NoError(t, m1Err)
	_, m2Err := makeTestMon()
	require.Error(t, m2Err)

	m1.Stop()
	_, m3Err := makeTestMon()
	require.NoError(t, m3Err)
}

func TestCheckInvalidConfig(t *testing.T) {
	serverMonConf := mockInvalidPluginConf(t)
	reg := mockPluginsReg()
	pipelineConnector := &MockPipelineConnector{}

	sched := scheduler.New(1)
	err := sched.Start()
	require.NoError(t, err)
	defer sched.Stop()

	m, err := newMonitor(serverMonConf, reg, pipelineConnector, sched, false, nil)
	// This could change if we decide the contract for newMonitor should always return a monitor
	require.Nil(t, m, "For this test to work we need a nil value for the monitor.")

	require.Error(t, checkMonitorConfig(serverMonConf, reg, false))
}
