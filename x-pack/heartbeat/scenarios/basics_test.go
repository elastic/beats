// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scenarios

import (
	"testing"

	"github.com/stretchr/testify/require"

	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/http"
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/icmp"
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/tcp"
	_ "github.com/elastic/beats/v7/x-pack/heartbeat/monitors/browser"
)

func TestSimpleScenariosBasicFields(t *testing.T) {
	Scenarios.RunAll(t, func(mtr *MonitorTestRun, err error) {
		require.GreaterOrEqual(t, len(mtr.Events()), 1)
		lastCg := ""
		for i, e := range mtr.Events() {
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
