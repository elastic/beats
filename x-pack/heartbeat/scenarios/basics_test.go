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

	"github.com/elastic/beats/v7/heartbeat/hbtestllext"
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/http"
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/icmp"
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/tcp"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/summarizer"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/summarizer/summarizertesthelper"
	"github.com/elastic/beats/v7/x-pack/heartbeat/scenarios/framework"
)

type CheckHistItem struct {
	cg      string
	summary *summarizer.JobSummary
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

			var summary *summarizer.JobSummary
			summaryIface, err := e.GetValue("summary")
			if err == nil {
				summary = summaryIface.(*summarizer.JobSummary)
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
	scenarioDB.RunTag(t, "lightweight", func(t *testing.T, mtr *framework.MonitorTestRun, err error) {
		all := mtr.Events()
		lastEvent, firstEvents := all[len(all)-1], all[:len(all)-1]
		testslike.Test(t,
			summarizertesthelper.SummaryValidator(1, 0),
			lastEvent.Fields)

		for _, e := range firstEvents {
			summary, _ := e.GetValue("summary")
			require.Nil(t, summary)
		}
	})
}

func TestBrowserSummaries(t *testing.T) {
	t.Parallel()
	scenarioDB.RunTag(t, "browser", func(t *testing.T, mtr *framework.MonitorTestRun, err error) {
		all := mtr.Events()
		lastEvent, firstEvents := all[len(all)-1], all[:len(all)-1]
		testslike.Test(t,
			summarizertesthelper.SummaryValidator(1, 0),
			lastEvent.Fields)

		for _, e := range firstEvents {
			summary, _ := e.GetValue("summary")
			require.Nil(t, summary)
		}
	})
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
