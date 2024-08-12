package guess

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInet4ProbeCorrectKretprobeBody(t *testing.T) {
	// test to make sure guessInetSockIPv4 produces a valid kprobe
	testGuess := guessInetSockIPv4{}

	probes, err := testGuess.Probes()
	require.NoError(t, err)

	for _, probe := range probes {
		probeArgs := strings.Split(probe.Probe.Fetchargs, " ")
		require.LessOrEqual(t, len(probeArgs), 128)
	}

}

func TestInet6ProbeCorrectKretprobeBody(t *testing.T) {
	// test to make sure guessInetSockIPv6 produces a valid kprobe
	testGuess6 := guessInetSockIPv6{}
	probes, err := testGuess6.Probes()
	require.NoError(t, err)

	for _, probe := range probes {
		probeArgs := strings.Split(probe.Probe.Fetchargs, " ")
		require.LessOrEqual(t, len(probeArgs), 128)
	}

}
