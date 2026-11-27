// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux || darwin || synthetics

package scenarios

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/testslike"

	"github.com/elastic/beats/v7/x-pack/heartbeat/scenarios/framework"
)

// TestAPISummaries runs a real inline apiJourney end-to-end through the agent.
func TestAPISummaries(t *testing.T) {
	t.Parallel()
	scenarioDB.RunTagWithSeparateTwists(t, "api", StdAttemptTwists, func(t *testing.T, mtr *framework.MonitorTestRun, err error) {
		require.NoError(t, err, "api scenario run must not error")
		all := mtr.Events()
		require.NotEmpty(t, all, "api journey must publish at least one event")
		lastEvent := all[len(all)-1]

		testslike.Test(t,
			lookslike.Compose(SummaryValidatorForStatus(mtr.Meta.Status)),
			lastEvent.Fields)

		var sawAPIType bool
		for _, e := range all {
			if v, gErr := e.GetValue("synthetics.journey.type"); gErr == nil && v == "api" {
				sawAPIType = true
				break
			}
		}
		require.True(t, sawAPIType, "at least one event must carry synthetics.journey.type: api")
	})
}
