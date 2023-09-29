// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scenarios

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/isdef"
	"github.com/elastic/go-lookslike/testslike"
	"github.com/elastic/go-lookslike/validator"

	"github.com/elastic/beats/v7/heartbeat/hbtest"
	"github.com/elastic/beats/v7/heartbeat/hbtestllext"
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/http"
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/icmp"
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/tcp"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/summarizer/jobsummary"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/summarizer/summarizertesthelper"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/heartbeat/scenarios/framework"
)

type CheckHistItem struct {
	cg      string
	summary *jobsummary.JobSummary
}

func TestSimpleScenariosBasicFields(t *testing.T) {
	t.Parallel()
	runner := func(t *testing.T, mtr *framework.MonitorTestRun, err error) {
		require.GreaterOrEqual(t, len(mtr.Events()), 1)
		var checkHist []*CheckHistItem
		for _, e := range mtr.Events() {
			testslike.Test(t, lookslike.MustCompile(map[string]interface{}{
				"monitor": map[string]interface{}{
					"id":          mtr.StdFields.ID,
					"name":        mtr.StdFields.Name,
					"type":        mtr.StdFields.Type,
					"check_group": isdef.IsString,
				},
			}), e.Fields)

			// Ensure that all check groups are equal and don't except across retries
			cgIface, err := e.GetValue("monitor.check_group")
			require.NoError(t, err)
			cg := cgIface.(string)

			var summary *jobsummary.JobSummary
			summaryIface, err := e.GetValue("summary")
			if err == nil {
				summary = summaryIface.(*jobsummary.JobSummary)
			}

			var lastCheck *CheckHistItem
			if len(checkHist) > 0 {
				lastCheck = checkHist[len(checkHist)-1]
			}

			curCheck := &CheckHistItem{cg: cg, summary: summary}

			checkHist = append(checkHist, curCheck)

			// If we have a prior check
			if lastCheck != nil {
				// If the last event was a summary, meaning this one is a retry
				if lastCheck.summary != nil {
					// then we expect a new check group
					require.NotEqual(t, lastCheck.cg, curCheck.cg)
				} else {
					// If we're within the same check due to multiple continuations
					// we expect equality
					require.Equal(t, lastCheck.cg, curCheck.cg)
				}
			}
		}
	}
	scenarioDB.RunAllWithSeparateTwists(t, []*framework.Twist{TwistMaxAttempts(2)}, runner)
}

func TestLightweightUrls(t *testing.T) {
	t.Parallel()
	scenarioDB.RunTag(t, "lightweight", func(t *testing.T, mtr *framework.MonitorTestRun, err error) {
		for _, e := range mtr.Events() {
			testslike.Test(t, lookslike.MustCompile(map[string]interface{}{
				"url": map[string]interface{}{
					"full":   isdef.IsNonEmptyString,
					"domain": isdef.IsNonEmptyString,
					"scheme": mtr.StdFields.Type,
				},
			}), e.Fields)
		}
	})
}

func TestLightweightSummaries(t *testing.T) {
	t.Parallel()
	scenarioDB.RunTagWithSeparateTwists(t, "lightweight", StdAttemptTwists, func(t *testing.T, mtr *framework.MonitorTestRun, err error) {
		all := mtr.Events()
		lastEvent := all[len(all)-1]
		testslike.Test(t,
			SummaryValidatorForStatus(mtr.Meta.Status),
			lastEvent.Fields)

		requireOneSummaryPerAttempt(t, all)
	})
}

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

func requireOneSummaryPerAttempt(t *testing.T, events []*beat.Event) {
	attemptCounter := uint16(1)
	// ensure we only have one summary per attempt
	for _, e := range events {
		summaryIface, _ := e.GetValue("summary")
		if summaryIface != nil {
			summary := summaryIface.(*jobsummary.JobSummary)
			require.Equal(t, attemptCounter, summary.Attempt)
			require.LessOrEqual(t, summary.Attempt, summary.MaxAttempts)
			attemptCounter++
		}
	}
}

func TestRunFromOverride(t *testing.T) {
	t.Parallel()
	scenarioDB.RunAllWithATwist(t, TwistAddRunFrom, func(t *testing.T, mtr *framework.MonitorTestRun, err error) {
		for idx, e := range mtr.Events() {
			stateIsDef := isdef.KeyMissing
			isLast := idx+1 == len(mtr.Events())
			if isLast {
				stateIsDef = hbtestllext.IsMonitorStateInLocation(TestLocationDefault.ID)
			}
			validator := lookslike.MustCompile(map[string]interface{}{
				"state": stateIsDef,
				"observer": map[string]interface{}{
					"name": TestLocationDefault.ID,
					"geo": map[string]interface{}{
						"name": TestLocationDefault.Geo.Name,
					},
				},
			})

			testslike.Test(t, validator, e.Fields)
		}
	})
}

func SummaryValidatorForStatus(ss monitorstate.StateStatus) validator.Validator {
	var expectedUp, expectedDown uint16 = 1, 0
	if ss == monitorstate.StatusDown {
		expectedUp, expectedDown = 0, 1
	}
	return summarizertesthelper.SummaryValidator(expectedUp, expectedDown)
}
