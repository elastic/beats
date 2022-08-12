package monitorstate

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTrackerRecord(t *testing.T) {
	monId := "mymonitor"
	mst := NewMonitorStateTracker(NilStateLoader)
	ms := mst.RecordStatus(monId, StatusUp)
	require.Equal(t, StatusUp, ms.Status)
	requireMSCounts(t, ms, 1, 0)

	for i := 0; i < FlappingThreshold; i++ {
		ms = mst.RecordStatus(monId, StatusUp)
	}
	require.Equal(t, StatusUp, ms.Status)
	requireMSCounts(t, ms, 4, 0)

	ms = mst.RecordStatus(monId, StatusDown)
	require.Equal(t, StatusDown, ms.Status)
	requireMSCounts(t, ms, 0, 1)
}
