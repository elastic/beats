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

	"github.com/elastic/beats/v7/heartbeat/hbtest"
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/http"
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/icmp"
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/tcp"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/summarizer/jobsummary"
	"github.com/elastic/beats/v7/x-pack/heartbeat/scenarios/framework"
)

func TestBrowserSummaries(t *testing.T) {
	t.Parallel()
	scenarioDB.RunTagWithSeparateTwists(t, "browser", StdAttemptTwists, func(t *testing.T, mtr *framework.MonitorTestRun, err error) {
		all := mtr.Events()
		lastEvent := all[len(all)-1]

		testslike.Test(t,
			lookslike.Compose(
				SummaryValidatorForStatus(mtr.Meta.Status),
				hbtest.URLChecks(t, mtr.Meta.URL),
			),
			lastEvent.Fields)

		monStatus, _ := lastEvent.GetValue("monitor.status")
		summaryIface, _ := lastEvent.GetValue("summary")
		summary := summaryIface.(*jobsummary.JobSummary)
		require.Equal(t, string(summary.Status), monStatus, "expected summary status and mon status to be equal in event: %v", lastEvent.Fields)

		requireOneSummaryPerAttempt(t, all)

	})
}
