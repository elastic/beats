package monitorstate

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func requireMSStatusCount(t *testing.T, ms *State, status StateStatus, count int) {
	if status == StatusUp {
		requireMSCounts(t, ms, count, 0)
	} else if status == StatusDown {
		requireMSCounts(t, ms, 0, count)
	} else {
		panic("can only check up or down statuses")
	}
}

func requireMSCounts(t *testing.T, ms *State, up int, down int) {
	require.Equal(t, up+down, ms.Checks)
	require.Equal(t, up, ms.Up)
	require.Equal(t, down, ms.Down)
}
