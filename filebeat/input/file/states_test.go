// +build !integration

package file

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var cleanupTests = []struct {
	title        string
	state        State
	countBefore  int
	cleanupCount int
	countAfter   int
}{
	{
		"Finished and TTL set to 0",
		State{
			TTL:      0,
			Finished: true,
		}, 1, 1, 0,
	},
	{
		"Unfinished but TTL set to 0",
		State{
			TTL:      0,
			Finished: false,
		}, 1, 0, 1,
	},
	{
		"TTL = -1 means not expiring",
		State{
			TTL:      -1,
			Finished: true,
		}, 1, 0, 1,
	},
	{
		"Expired and finished",
		State{
			TTL:       1 * time.Second,
			Timestamp: time.Now().Add(-2 * time.Second),
			Finished:  true,
		}, 1, 1, 0,
	},
	{
		"Expired but unfinished",
		State{
			TTL:       1 * time.Second,
			Timestamp: time.Now().Add(-2 * time.Second),
			Finished:  false,
		}, 1, 0, 1,
	},
}

func TestCleanup(t *testing.T) {
	for _, test := range cleanupTests {
		test := test
		t.Run(test.title, func(t *testing.T) {
			states := NewStates()
			states.SetStates([]State{test.state})

			assert.Equal(t, test.countBefore, states.Count())
			cleanupCount, _ := states.Cleanup()
			assert.Equal(t, test.cleanupCount, cleanupCount)
			assert.Equal(t, test.countAfter, states.Count())
		})
	}
}
