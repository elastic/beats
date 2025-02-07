// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package maintwin

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/teambition/rrule-go"
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
				Freq:      3,
				Dtstart:   time.Now().Format(time.RFC3339),
				Duration:  mustParseDuration("2h"),
				Byweekday: []string{"SU", "MO", "TU", "WE", "TH", "FR", "SA"},
				Count:     10,
			},
			// add 30 minutes, 1 hour, 1 hour 30 minutes to the start time
			[]string{time.Now().Add(30 * time.Minute).Format(time.RFC3339), time.Now().Add(60 * time.Minute).Format(time.RFC3339), time.Now().Add(90 * time.Minute).Format(time.RFC3339)},
			[]string{time.Now().Add(180 * time.Minute).Format(time.RFC3339), time.Now().Add(540 * time.Minute).Format(time.RFC3339)},
		},

		{
			name: "Daily maintenance window for 2 hours",
			mw: MaintWin{
				Freq:     rrule.DAILY,
				Dtstart:  "2025-02-06T21:00:00Z",
				Duration: mustParseDuration("2h"),
			},
			positiveMatches: []string{"2025-02-06T21:30:00Z", "2025-02-06T22:45:00Z"},
			negativeMatches: []string{"2025-02-06T23:01:00Z", "2025-02-07T00:00:00Z"},
		},

		{
			name: "Monthly maintenance window on the 1st",
			mw: MaintWin{
				Freq:       rrule.MONTHLY,
				Dtstart:    "2025-02-01T10:00:00Z",
				Duration:   mustParseDuration("2h"),
				Bymonthday: []int{1},
			},
			positiveMatches: []string{"2025-03-01T10:30:00Z", "2025-04-01T11:45:00Z"},
			negativeMatches: []string{"2025-02-02T10:30:00Z", "2025-02-01T12:01:00Z"},
		},

		{
			name: "Weekly on Monday and Wednesday from 8 AM to 10 AM",
			mw: MaintWin{
				Freq:      rrule.WEEKLY,
				Dtstart:   "2025-02-03T08:00:00Z",
				Duration:  mustParseDuration("2h"),
				Byweekday: []string{"MO", "WE"},
			},
			positiveMatches: []string{"2025-02-10T09:30:00Z", "2025-02-12T08:15:00Z"},
			negativeMatches: []string{"2025-02-10T10:30:00Z", "2025-02-11T09:30:00Z"},
		},

		{
			name: "Hourly maintenance for 15 minutes",
			mw: MaintWin{
				Freq:     rrule.HOURLY,
				Dtstart:  "2025-02-06T00:00:00Z",
				Duration: mustParseDuration("15m"),
			},
			positiveMatches: []string{"2025-02-06T00:05:00Z", "2025-02-06T01:10:00Z"},
			negativeMatches: []string{"2025-02-06T00:16:00Z", "2025-02-06T01:30:00Z"},
		},

		{
			name: "First Friday of every month",
			mw: MaintWin{
				Freq:      rrule.MONTHLY,
				Dtstart:   "2025-02-07T12:00:00Z",
				Duration:  mustParseDuration("2h"),
				Byweekday: []string{"FR"},
				Bysetpos:  []int{1}, // First Friday of the month
			},
			positiveMatches: []string{"2025-03-07T12:30:00Z"},
			negativeMatches: []string{"2025-02-14T12:30:00Z", "2025-04-14T13:00:00Z"},
		},

		{
			name: "Every Saturday and Sunday from 5 PM to 8 PM",
			mw: MaintWin{
				Freq:      rrule.WEEKLY,
				Dtstart:   "2025-02-08T17:00:00Z",
				Duration:  mustParseDuration("3h"),
				Byweekday: []string{"SA", "SU"},
			},
			positiveMatches: []string{"2025-02-09T18:30:00Z", "2025-02-15T19:00:00Z"},
			negativeMatches: []string{"2025-02-09T20:30:00Z", "2025-02-10T17:30:00Z"},
		},

		{
			name: "Monthly on the 15th from 6 AM to 9 AM",
			mw: MaintWin{
				Freq:       rrule.MONTHLY,
				Dtstart:    "2025-02-15T06:00:00Z",
				Duration:   mustParseDuration("3h"),
				Bymonthday: []int{15},
			},
			positiveMatches: []string{"2025-03-15T07:30:00Z", "2025-04-15T08:45:00Z"},
			negativeMatches: []string{"2025-02-16T07:30:00Z", "2025-02-15T09:30:00Z"},
		},

		{
			name: "Yearly maintenance on Jan 1 from Midnight to 3 AM",
			mw: MaintWin{
				Freq:       rrule.YEARLY,
				Dtstart:    "2025-01-01T00:00:00Z",
				Duration:   mustParseDuration("3h"),
				Bymonthday: []int{1},
			},
			positiveMatches: []string{"2026-01-01T01:30:00Z", "2027-01-01T02:45:00Z"},
			negativeMatches: []string{"2025-01-02T01:30:00Z", "2025-01-01T03:30:00Z"},
		},

		{
			name: "Every other day for 4 hours",
			mw: MaintWin{
				Freq:     rrule.DAILY,
				Dtstart:  "2025-02-06T08:00:00Z",
				Duration: mustParseDuration("4h"),
				Interval: 2, // Every other day
				Count:    10,
			},
			positiveMatches: []string{"2025-02-08T09:30:00Z", "2025-02-10T11:00:00Z"},
			negativeMatches: []string{"2025-02-07T09:30:00Z", "2025-02-06T13:00:00Z"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rules := []*rrule.RRule{}
			r, err := c.mw.Parse()
			require.NoError(t, err)
			rules = append(rules, r)
			durations := []time.Duration{c.mw.Duration}
			pmw := ParsedMaintWin{Rules: rules, Durations: durations}
			for _, m := range c.positiveMatches {
				t.Run(fmt.Sprintf("does match %s", m), func(t *testing.T) {
					pt, err := time.Parse(time.RFC3339, m)
					require.NoError(t, err)
					assert.True(t, pmw.IsActive(pt.UTC()))
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
