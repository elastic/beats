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
	"github.com/elastic/beats/v7/x-pack/heartbeat/scenarios/framework"
)

func TestSimpleScenariosBasicFields(t *testing.T) {
	scenarioDB.RunAll(t, func(t *testing.T, mtr *framework.MonitorTestRun, err error) {
		require.GreaterOrEqual(t, len(mtr.Events()), 1)
		lastCg := ""
		for i, e := range mtr.Events() {
			testslike.Test(t, lookslike.MustCompile(map[string]interface{}{
				"monitor": map[string]interface{}{
					"id":          mtr.StdFields.ID,
					"name":        mtr.StdFields.Name,
					"type":        mtr.StdFields.Type,
					"check_group": isdef.IsString,
				},
			}), e.Fields)

			// Ensure that all check groups are equal and don't change
			cg, err := e.GetValue("monitor.check_group")
			require.NoError(t, err)
			cgStr := cg.(string)
			if i == 0 {
				lastCg = cgStr
			} else {
				require.Equal(t, lastCg, cgStr)
			}
		}
	})
}

func TestLightweightUrls(t *testing.T) {
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
	scenarioDB.RunTag(t, "lightweight", func(t *testing.T, mtr *framework.MonitorTestRun, err error) {
		all := mtr.Events()
		lastEvent, firstEvents := all[len(all)-1], all[:len(all)-1]
		testslike.Test(t, lookslike.MustCompile(map[string]interface{}{
			"summary": map[string]interface{}{
				"up":   hbtestllext.IsUint16,
				"down": hbtestllext.IsUint16,
			},
		}), lastEvent.Fields)

		for _, e := range firstEvents {
			summary, _ := e.GetValue("summary")
			require.Nil(t, summary)
		}
	})
}

func TestRunFromOverride(t *testing.T) {
	scenarioDB.RunAllWithATwist(t, TwistAddRunFrom, func(t *testing.T, mtr *framework.MonitorTestRun, err error) {
		for _, e := range mtr.Events() {
			testslike.Test(t, lookslike.MustCompile(map[string]interface{}{
				"state": hbtestllext.IsMonitorStateInLocation(TestLocationDefault.ID),
				"observer": map[string]interface{}{
					"name": TestLocationDefault.ID,
					"geo": map[string]interface{}{
						"name": TestLocationDefault.Geo.Name,
					},
				},
			}), e.Fields)
		}
	})
}
