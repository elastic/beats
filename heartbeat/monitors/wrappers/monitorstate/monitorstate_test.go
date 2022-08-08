package monitorstate

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIsFlappingWithFlapHistory(t *testing.T) {
	ms := newMonitorState("test", StatusUp)
	require.False(t, ms.isFlapping())

	ms.FlapHistory = []HistoricalStatus{
		{TsMs: 0, Status: StatusUp},
		{TsMs: 1, Status: StatusDown},
		{TsMs: 2, Status: StatusDown},
	}

	require.True(t, ms.isFlapping())
}

func TestRecordingChecks(t *testing.T) {
	ms := newMonitorState("test", StatusUp)

	require.Equal(t, 1, ms.Checks)
	require.Equal(t, 1, ms.Up)
	require.Equal(t, 0, ms.Down)

	ms.recordCheck(StatusUp)
	ms.recordCheck(StatusUp)
	ms.recordCheck(StatusDown)

	require.Equal(t, 4, ms.Checks)
	require.Equal(t, 3, ms.Up)
	require.Equal(t, 1, ms.Down)
}

func TestWouldStatusEndFlapping(t *testing.T) {
	ms := newMonitorState("test", StatusUp)

	ms.FlapHistory = []HistoricalStatus{
		{TsMs: 0, Status: StatusUp},
		{TsMs: 1000, Status: StatusDown},
		{TsMs: 2000, Status: StatusDown},
	}
}
