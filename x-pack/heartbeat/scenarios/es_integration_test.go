// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scenarios

import (
	"testing"

	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
	"github.com/stretchr/testify/require"
)

func TestBlankState(t *testing.T) {
	Scenarios.RunAllWithATwist(t, TwistAddLocation, func(t *testing.T, mtr *MonitorTestRun, err error) {
		for _, e := range mtr.Events() {
			if stateIface, _ := e.Fields.GetValue("state"); stateIface != nil {
				state, ok := stateIface.(*monitorstate.State)
				require.True(t, ok, "state is not a monitorstate.State, got %v", state)

				require.Equal(t, monitorstate.StatusUp, state.Status)
			}
		}
	})
}
