package monitorstate

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRecordingAndFlapping(t *testing.T) {
	ms := newMonitorState("test", StatusUp)
	recordFlappingSeries(ms)
	require.Equal(t, StatusFlapping, ms.Status)
	require.Equal(t, FlappingThreshold+1, ms.Checks)
	require.Equal(t, ms.Up+ms.Down, ms.Checks)

	// Use double the flapping threshold so any transitions after this are stable
	priorChecksCount := ms.Checks
	recordStableSeries(ms, FlappingThreshold*2, StatusDown)
	require.Equal(t, StatusDown, ms.Status)
	// The count should be FlappingThreshold+1 since we used double the threshold before
	// This is because we have one full threshold of stable checks, as well as the final check that
	// flipped us out of the threshold, which goes toward the new state.
	require.Equal(t, FlappingThreshold+1, ms.Checks)
	require.Equal(t, 0, ms.Up)
	require.Equal(t, FlappingThreshold+1, ms.Down)
	require.Equal(t, priorChecksCount+FlappingThreshold-1, ms.Ends.Checks)

	// Since we're now in a stable state a single up check should create a new state from a stable one
	ms.recordCheck(StatusUp)
	require.Equal(t, StatusUp, ms.Status)
	require.Equal(t, 1, ms.Checks)
	require.Equal(t, 1, ms.Up)
	require.Equal(t, 0, ms.Down)
}

// recordFlappingSeries is a helper that should always put the monitor into a flapping state.
func recordFlappingSeries(ms *MonitorState) {
	for i := 0; i < FlappingThreshold; i++ {
		if i%2 == 0 {
			ms.recordCheck(StatusUp)
		} else {
			ms.recordCheck(StatusDown)
		}
	}
}

// recordStableSeries is a test helper for repeatedly recording one status
func recordStableSeries(ms *MonitorState, count int, s StateStatus) {
	for i := 0; i < count; i++ {
		ms.recordCheck(s)
	}
}
