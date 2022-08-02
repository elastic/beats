package scenarios

import (
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/http"
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/icmp"
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/tcp"
	_ "github.com/elastic/beats/v7/x-pack/heartbeat/monitors/browser"
	"github.com/stretchr/testify/require"
	"testing"
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
