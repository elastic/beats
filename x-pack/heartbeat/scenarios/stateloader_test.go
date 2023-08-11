// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scenarios

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
	"github.com/elastic/beats/v7/x-pack/heartbeat/scenarios/framework"
)

var esIntegTwists = framework.MultiTwist(TwistAddRunFrom, TwistMultiRun(3))

func TestStateContinuity(t *testing.T) {
	t.Parallel()
	scenarioDB.RunOneWithATwist(t, esIntegTwists, func(t *testing.T, mtr *framework.MonitorTestRun, err error) {
		lastSS := framework.LastState(mtr.Events())

		assert.Equal(t, monitorstate.StatusUp, lastSS.State.Status)

		allSS := framework.AllStates(mtr.Events())
		assert.Len(t, allSS, 3)

		assert.Equal(t, 3, lastSS.State.Checks)
	})
}
