package monitorstate

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func requireMSCounts(t *testing.T, ms *MonitorState, up int, down int) {
	require.Equal(t, up+down, ms.Checks)
	require.Equal(t, up, ms.Up)
	require.Equal(t, down, ms.Down)
}
