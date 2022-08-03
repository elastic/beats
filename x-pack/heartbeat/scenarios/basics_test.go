// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scenarios

import (
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/http"
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/icmp"
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/tcp"
	_ "github.com/elastic/beats/v7/x-pack/heartbeat/monitors/browser"
	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/isdef"
	"github.com/elastic/go-lookslike/testslike"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSimpleScenariosBasicFields(t *testing.T) {
	Scenarios.RunAll(t, func(mtr *MonitorTestRun, err error) {
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

func TestLightweightSummaries(t *testing.T) {
	Scenarios.RunTag(t, "lightweight", func(mtr *MonitorTestRun, err error) {
		require.GreaterOrEqual(t, len(mtr.Events()), 1)
		lastCg := ""

		for i, e := range mtr.Events() {
			testslike.Test(t, lookslike.MustCompile(map[string]interface{}{
				"url": map[string]interface{}{
					"full":   isdef.IsNonEmptyString,
					"domain": isdef.IsNonEmptyString,
					"scheme": mtr.StdFields.Type,
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
