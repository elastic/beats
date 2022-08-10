package monitorstate

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTrackerRecord(t *testing.T) {
	monId := "mymonitor"
	mst := NewMonitorStateTracker()
	ms := mst.RecordStatus(monId, StatusUp)
	require.Equal(t, StatusUp, ms.Status)
	require.Equal(t, 1, ms.Checks)
	require.Equal(t, 1, ms.Up)
	require.Equal(t, 0, ms.Down)

	for i := 0; i < FlappingThreshold; i++ {
		ms = mst.RecordStatus(monId, StatusUp)
	}
	require.Equal(t, StatusUp, ms.Status)
	require.Equal(t, 4, ms.Checks)
	require.Equal(t, 4, ms.Up)
	require.Equal(t, 0, ms.Down)

	ms = mst.RecordStatus(monId, StatusDown)
	require.Equal(t, StatusDown, ms.Status)
	require.Equal(t, 1, ms.Checks)
	require.Equal(t, 0, ms.Up)
	require.Equal(t, 1, ms.Down)
}
