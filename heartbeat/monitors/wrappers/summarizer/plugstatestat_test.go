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

package summarizer

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// TestStateStatusPluginBeforeEachEventSetsCheckGroup verifies that the state
// status plugins write monitor.check_group on the event before the job runs.
// This is what lets synthetics monitors propagate the check group to the runner
// as the APM trace id so journey executions can be cross-linked with APM data.
func TestStateStatusPluginBeforeEachEventSetsCheckGroup(t *testing.T) {
	tracker := monitorstate.NewTracker(monitorstate.NilStateLoader, false)
	sf := stdfields.StdMonitorFields{ID: "testmon", Name: "testmon", Type: "browser", MaxAttempts: 1}

	t.Run("browser", func(t *testing.T) {
		p := NewBrowserStateStatusplugin(tracker, sf)
		event := &beat.Event{Fields: mapstr.M{}}
		p.BeforeEachEvent(event)
		cg, err := event.GetValue("monitor.check_group")
		require.NoError(t, err)
		require.Equal(t, p.cssp.checkGroup+"-1", cg)
	})

	t.Run("lightweight", func(t *testing.T) {
		p := NewLightweightStateStatusPlugin(tracker, sf)
		event := &beat.Event{Fields: mapstr.M{}}
		p.BeforeEachEvent(event)
		cg, err := event.GetValue("monitor.check_group")
		require.NoError(t, err)
		require.Equal(t, p.cssp.checkGroup+"-1", cg)
	})
}
