// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scenarios

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
	"github.com/elastic/beats/v7/libbeat/beat"
)

type stateEvent struct {
	event *beat.Event
	state *monitorstate.State
}

func allStates(t *testing.T, events []*beat.Event) (stateEvents []stateEvent) {
	for _, e := range events {
		if stateIface, _ := e.Fields.GetValue("state"); stateIface != nil {
			state, ok := stateIface.(*monitorstate.State)
			require.True(t, ok, "state is not a monitorstate.State, got %v", state)

			se := stateEvent{event: e, state: state}
			stateEvents = append(stateEvents, se)
		}
	}
	return stateEvents
}

func lastState(t *testing.T, events []*beat.Event) stateEvent {
	all := allStates(t, events)
	require.NotEmpty(t, all)
	return all[len(all)-1]
}

var esIntegTwists = MultiTwist(TwistAddLocation, TwistMultiRun(3), TwistEnableES)

func TestStateContinuity(t *testing.T) {
	Scenarios.RunAllWithATwist(t, esIntegTwists, func(t *testing.T, mtr *MonitorTestRun, err error) {
		lastSS := lastState(t, mtr.Events())

		require.Equal(t, monitorstate.StatusUp, lastSS.state.Status)

		allSS := allStates(t, mtr.Events())
		require.Len(t, allSS, 3)

		require.Equal(t, 3, lastSS.state.Checks)
	})
}
