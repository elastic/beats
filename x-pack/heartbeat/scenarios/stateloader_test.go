// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scenarios

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/heartbeat/scenarios/framework"
)

const numRuns = 2

var esIntegTwists = framework.MultiTwist(TwistAddRunFrom, TwistMultiRun(numRuns))

func TestStateContinuity(t *testing.T) {
	t.Parallel()
	scenarioDB.RunAllWithATwist(t, esIntegTwists, func(t *testing.T, mtr *framework.MonitorTestRun, err error) {
		events := mtr.Events()
		var errors = []*beat.Event{}
		var sout string
		for _, e := range events {
			if message, ok := e.GetValue("synthetics.payload.message"); ok == nil {
				sout = sout + "\n" + message.(string)
			}
			if _, ok := e.GetValue("error"); ok == nil {
				errors = append(errors, e)
			}
		}

		lastSS := framework.LastState(mtr.Events())

		assert.Equal(t, mtr.Meta.Status, lastSS.State.Status, "monitor had unexpected state %v, synthetics console output: %s, errors", lastSS.State.Status, sout, errors)

		allSS := framework.AllStates(mtr.Events())
		assert.Len(t, allSS, numRuns)

		assert.Equal(t, numRuns, lastSS.State.Checks)
	})
}
