package monitorstate

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRecordingAndFlapping(t *testing.T) {
	monitorID := "test"
	ms := newMonitorState(monitorID, StatusUp)
	recordFlappingSeries(monitorID, ms)
	require.Equal(t, StatusFlapping, ms.Status)
	require.Equal(t, FlappingThreshold+1, ms.Checks)
	require.Equal(t, ms.Up+ms.Down, ms.Checks)

	// Use double the flapping threshold so any transitions after this are stable
	priorChecksCount := ms.Checks
	recordStableSeries(monitorID, ms, FlappingThreshold*2, StatusDown)
	require.Equal(t, StatusDown, ms.Status)
	// The count should be FlappingThreshold+1 since we used double the threshold before
	// This is because we have one full threshold of stable checks, as well as the final check that
	// flipped us out of the threshold, which goes toward the new state.
	requireMSCounts(t, ms, 0, FlappingThreshold+1)
	require.Equal(t, priorChecksCount+FlappingThreshold-1, ms.Ends.Checks)

	// Since we're now in a stable state a single up check should create a new state from a stable one
	ms.recordCheck(monitorID, StatusUp)
	require.Equal(t, StatusUp, ms.Status)
	requireMSCounts(t, ms, 1, 0)
}

// recordFlappingSeries is a helper that should always put the monitor into a flapping state.
func recordFlappingSeries(monitorID string, ms *State) {
	for i := 0; i < FlappingThreshold; i++ {
		if i%2 == 0 {
			ms.recordCheck(monitorID, StatusUp)
		} else {
			ms.recordCheck(monitorID, StatusDown)
		}
	}
}

// recordStableSeries is a test helper for repeatedly recording one status
func recordStableSeries(monitorID string, ms *State, count int, s StateStatus) {
	for i := 0; i < count; i++ {
		ms.recordCheck(monitorID, s)
	}
}
