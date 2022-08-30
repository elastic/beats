// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scenarios

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
)

var esIntegTwists = MultiTwist(TwistAddLocation, TwistMultiRun(3))

func TestStateContinuity(t *testing.T) {
	Scenarios.RunAllWithATwist(t, esIntegTwists, func(t *testing.T, mtr *MonitorTestRun, err error) {
		lastSS := LastState(mtr.Events())

		require.Equal(t, monitorstate.StatusUp, lastSS.state.Status)

		allSS := AllStates(mtr.Events())
		require.Len(t, allSS, 3)

		require.Equal(t, 3, lastSS.state.Checks)
	})
}
