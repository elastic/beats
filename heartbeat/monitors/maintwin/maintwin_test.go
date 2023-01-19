package maintwin

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaintWin(t *testing.T) {
	cases := []struct {
		name            string
		mw              MaintWin
		positiveMatches []string
		negativeMatches []string
	}{
		{
			"Every sunday at midnight to 1 AM",
			MaintWin{
				Zone:     "UTC",
				Start:    "0 0 * * SUN *",
				Duration: mustParseDuration("1h"),
			},
			[]string{"2023-01-15T00:00:00+00:00", "2023-01-15T00:00:01+00:00", "2023-01-15T01:00:00+00:00"},
			[]string{"2023-01-15T01:00:01+00:00", "2023-01-19T02:00:32+00:00"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pmw, err := c.mw.Parse()
			require.NoError(t, err)
			for _, m := range c.positiveMatches {
				t.Run(fmt.Sprintf("does match %s", m), func(t *testing.T) {
					pt, err := time.Parse(time.RFC3339, m)
					require.NoError(t, err)
					assert.True(t, pmw.IsActive(pt))
				})
			}
			for _, m := range c.negativeMatches {
				t.Run(fmt.Sprintf("does not match %s", m), func(t *testing.T) {
					pt, err := time.Parse(time.RFC3339, m)
					require.NoError(t, err)
					assert.False(t, pmw.IsActive(pt))
				})
			}
		})
	}
}

func mustParseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(fmt.Sprintf("could not parse duration %s: %s", s, err))
	}
	return d
}
