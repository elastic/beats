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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Occurrences come from Kibana @kbn/rrule snapshots in
// src/platform/packages/shared/kbn-rrule/rrule.test.ts — the library Kibana
// uses to decide whether a maintenance window is running.
func TestKibanaRRuleParity(t *testing.T) {
	cases := []struct {
		name    string
		mw      MaintWin
		wantISO []string
	}{
		{
			name: "yearly interval 1",
			mw: MaintWin{
				Freq:     "yearly",
				Dtstart:  "2019-01-01T00:00:00.000Z",
				Tzid:     "UTC",
				Interval: 1,
				Duration: time.Hour,
				Count:    10,
			},
			wantISO: []string{
				"2019-01-01T00:00:00Z",
				"2020-01-01T00:00:00Z",
				"2021-01-01T00:00:00Z",
				"2022-01-01T00:00:00Z",
				"2023-01-01T00:00:00Z",
				"2024-01-01T00:00:00Z",
				"2025-01-01T00:00:00Z",
				"2026-01-01T00:00:00Z",
				"2027-01-01T00:00:00Z",
				"2028-01-01T00:00:00Z",
			},
		},
		{
			name: "monthly interval 6 with bymonthday",
			mw: MaintWin{
				Freq:       "monthly",
				Dtstart:    "2019-01-01T00:00:00.000Z",
				Tzid:       "UTC",
				Interval:   6,
				Duration:   time.Hour,
				Bymonthday: []int{10, 20},
				Count:      6,
			},
			wantISO: []string{
				"2019-01-10T00:00:00Z",
				"2019-01-20T00:00:00Z",
				"2019-07-10T00:00:00Z",
				"2019-07-20T00:00:00Z",
				"2020-01-10T00:00:00Z",
				"2020-01-20T00:00:00Z",
			},
		},
		{
			name: "weekly interval 2",
			mw: MaintWin{
				Freq:     "weekly",
				Dtstart:  "2019-12-19T00:00:00.000Z",
				Tzid:     "UTC",
				Interval: 2,
				Duration: time.Hour,
				Count:    14,
			},
			wantISO: []string{
				"2019-12-19T00:00:00Z",
				"2020-01-02T00:00:00Z",
				"2020-01-16T00:00:00Z",
				"2020-01-30T00:00:00Z",
				"2020-02-13T00:00:00Z",
				"2020-02-27T00:00:00Z",
				"2020-03-12T00:00:00Z",
				"2020-03-26T00:00:00Z",
				"2020-04-09T00:00:00Z",
				"2020-04-23T00:00:00Z",
				"2020-05-07T00:00:00Z",
				"2020-05-21T00:00:00Z",
				"2020-06-04T00:00:00Z",
				"2020-06-18T00:00:00Z",
			},
		},
		{
			name: "until is exclusive of the bound (Kibana uses current >= until)",
			mw: MaintWin{
				Freq:     "monthly",
				Dtstart:  "2019-01-01T00:00:00.000Z",
				Tzid:     "UTC",
				Interval: 1,
				Duration: time.Hour,
				Until:    "2019-12-01T00:00:00.000Z",
			},
			wantISO: []string{
				"2019-01-01T00:00:00Z",
				"2019-02-01T00:00:00Z",
				"2019-03-01T00:00:00Z",
				"2019-04-01T00:00:00Z",
				"2019-05-01T00:00:00Z",
				"2019-06-01T00:00:00Z",
				"2019-07-01T00:00:00Z",
				"2019-08-01T00:00:00Z",
				"2019-09-01T00:00:00Z",
				"2019-10-01T00:00:00Z",
				"2019-11-01T00:00:00Z",
			},
		},
		{
			name: "one-shot yearly count 1 as Kibana UI stores non-recurring windows",
			mw: MaintWin{
				Freq:     "yearly",
				Dtstart:  "2025-06-10T11:40:29.124Z",
				Tzid:     "Europe/Berlin",
				Count:    1,
				Duration: 30 * time.Minute,
			},
			wantISO: []string{"2025-06-10T11:40:29Z"},
		},
		{
			name: "weekly SA/SU/MO",
			mw: MaintWin{
				Freq:      "weekly",
				Dtstart:   "2019-01-01T00:00:00.000Z",
				Tzid:      "UTC",
				Interval:  1,
				Duration:  time.Hour,
				Byweekday: []string{"SA", "SU", "MO"},
				Count:     9,
			},
			wantISO: []string{
				"2019-01-05T00:00:00Z",
				"2019-01-06T00:00:00Z",
				"2019-01-07T00:00:00Z",
				"2019-01-12T00:00:00Z",
				"2019-01-13T00:00:00Z",
				"2019-01-14T00:00:00Z",
				"2019-01-19T00:00:00Z",
				"2019-01-20T00:00:00Z",
				"2019-01-21T00:00:00Z",
			},
		},
		{
			name: "monthly nth weekdays +1TU +2TU -1FR -2FR",
			mw: MaintWin{
				Freq:      "monthly",
				Dtstart:   "2023-01-01T00:00:00.000Z",
				Tzid:      "UTC",
				Interval:  1,
				Duration:  time.Hour,
				Byweekday: []string{"+1TU", "+2TU", "-1FR", "-2FR"},
				Count:     12,
			},
			wantISO: []string{
				"2023-01-03T00:00:00Z",
				"2023-01-10T00:00:00Z",
				"2023-01-20T00:00:00Z",
				"2023-01-27T00:00:00Z",
				"2023-02-07T00:00:00Z",
				"2023-02-14T00:00:00Z",
				"2023-02-17T00:00:00Z",
				"2023-02-24T00:00:00Z",
				"2023-03-07T00:00:00Z",
				"2023-03-14T00:00:00Z",
				"2023-03-24T00:00:00Z",
				"2023-03-31T00:00:00Z",
			},
		},
		{
			name: "weekly Saturday in Europe/Madrid from Friday 23:00 UTC",
			mw: MaintWin{
				Freq:      "weekly",
				Dtstart:   "2023-01-06T23:00:00Z",
				Tzid:      "Europe/Madrid",
				Interval:  1,
				Duration:  time.Hour,
				Byweekday: []string{"SA"},
				Count:     12,
			},
			wantISO: []string{
				"2023-01-06T23:00:00Z",
				"2023-01-13T23:00:00Z",
				"2023-01-20T23:00:00Z",
				"2023-01-27T23:00:00Z",
				"2023-02-03T23:00:00Z",
				"2023-02-10T23:00:00Z",
				"2023-02-17T23:00:00Z",
				"2023-02-24T23:00:00Z",
				"2023-03-03T23:00:00Z",
				"2023-03-10T23:00:00Z",
				"2023-03-17T23:00:00Z",
				"2023-03-24T23:00:00Z",
			},
		},
		{
			name: "weekly Saturday in UTC from Friday 23:00 UTC starts the following Saturday",
			mw: MaintWin{
				Freq:      "weekly",
				Dtstart:   "2023-01-06T23:00:00Z",
				Tzid:      "UTC",
				Interval:  1,
				Duration:  time.Hour,
				Byweekday: []string{"SA"},
				Count:     12,
			},
			wantISO: []string{
				"2023-01-07T23:00:00Z",
				"2023-01-14T23:00:00Z",
				"2023-01-21T23:00:00Z",
				"2023-01-28T23:00:00Z",
				"2023-02-04T23:00:00Z",
				"2023-02-11T23:00:00Z",
				"2023-02-18T23:00:00Z",
				"2023-02-25T23:00:00Z",
				"2023-03-04T23:00:00Z",
				"2023-03-11T23:00:00Z",
				"2023-03-18T23:00:00Z",
				"2023-03-25T23:00:00Z",
			},
		},
		{
			name: "yearly bymonth Feb and May",
			mw: MaintWin{
				Freq:     "yearly",
				Dtstart:  "2019-12-19T00:00:00.000Z",
				Tzid:     "UTC",
				Interval: 1,
				Duration: time.Hour,
				Bymonth:  []int{2, 5},
				Count:    14,
			},
			wantISO: []string{
				"2020-02-19T00:00:00Z",
				"2020-05-19T00:00:00Z",
				"2021-02-19T00:00:00Z",
				"2021-05-19T00:00:00Z",
				"2022-02-19T00:00:00Z",
				"2022-05-19T00:00:00Z",
				"2023-02-19T00:00:00Z",
				"2023-05-19T00:00:00Z",
				"2024-02-19T00:00:00Z",
				"2024-05-19T00:00:00Z",
				"2025-02-19T00:00:00Z",
				"2025-05-19T00:00:00Z",
				"2026-02-19T00:00:00Z",
				"2026-05-19T00:00:00Z",
			},
		},
		{
			name: "monthly bymonthday 1 and 15",
			mw: MaintWin{
				Freq:       "monthly",
				Dtstart:    "2019-12-19T00:00:00.000Z",
				Tzid:       "UTC",
				Interval:   1,
				Duration:   time.Hour,
				Bymonthday: []int{1, 15},
				Count:      14,
			},
			wantISO: []string{
				"2020-01-01T00:00:00Z",
				"2020-01-15T00:00:00Z",
				"2020-02-01T00:00:00Z",
				"2020-02-15T00:00:00Z",
				"2020-03-01T00:00:00Z",
				"2020-03-15T00:00:00Z",
				"2020-04-01T00:00:00Z",
				"2020-04-15T00:00:00Z",
				"2020-05-01T00:00:00Z",
				"2020-05-15T00:00:00Z",
				"2020-06-01T00:00:00Z",
				"2020-06-15T00:00:00Z",
				"2020-07-01T00:00:00Z",
				"2020-07-15T00:00:00Z",
			},
		},
		{
			name: "yearly bymonth and bymonthday",
			mw: MaintWin{
				Freq:       "yearly",
				Dtstart:    "2019-12-19T00:00:00.000Z",
				Tzid:       "UTC",
				Interval:   1,
				Duration:   time.Hour,
				Bymonth:    []int{2, 5},
				Bymonthday: []int{8},
				Count:      14,
			},
			wantISO: []string{
				"2020-02-08T00:00:00Z",
				"2020-05-08T00:00:00Z",
				"2021-02-08T00:00:00Z",
				"2021-05-08T00:00:00Z",
				"2022-02-08T00:00:00Z",
				"2022-05-08T00:00:00Z",
				"2023-02-08T00:00:00Z",
				"2023-05-08T00:00:00Z",
				"2024-02-08T00:00:00Z",
				"2024-05-08T00:00:00Z",
				"2025-02-08T00:00:00Z",
				"2025-05-08T00:00:00Z",
				"2026-02-08T00:00:00Z",
				"2026-05-08T00:00:00Z",
			},
		},
		{
			name: "Kibana UI daily with byweekday WE",
			mw: MaintWin{
				Freq:      "daily",
				Dtstart:   "2023-03-22T00:00:00.000Z",
				Tzid:      "UTC",
				Interval:  1,
				Duration:  48 * time.Hour,
				Byweekday: []string{"WE"},
				Count:     4,
			},
			wantISO: []string{
				"2023-03-22T00:00:00Z",
				"2023-03-29T00:00:00Z",
				"2023-04-05T00:00:00Z",
				"2023-04-12T00:00:00Z",
			},
		},
		{
			name: "hourly interval 1 from Kibana @kbn/rrule",
			mw: MaintWin{
				Freq:     "hourly",
				Dtstart:  "2019-01-01T00:00:00.000Z",
				Tzid:     "UTC",
				Interval: 1,
				Duration: time.Hour,
				Count:    10,
			},
			wantISO: []string{
				"2019-01-01T00:00:00Z",
				"2019-01-01T01:00:00Z",
				"2019-01-01T02:00:00Z",
				"2019-01-01T03:00:00Z",
				"2019-01-01T04:00:00Z",
				"2019-01-01T05:00:00Z",
				"2019-01-01T06:00:00Z",
				"2019-01-01T07:00:00Z",
				"2019-01-01T08:00:00Z",
				"2019-01-01T09:00:00Z",
			},
		},
		{
			name: "Kibana UI monthly default +4WE from convertToRRule",
			mw: MaintWin{
				Freq:      "monthly",
				Dtstart:   "2023-03-22T00:00:00.000Z",
				Tzid:      "UTC",
				Interval:  1,
				Duration:  time.Hour,
				Byweekday: []string{"+4WE"},
				Count:     4,
			},
			wantISO: []string{
				"2023-03-22T00:00:00Z",
				"2023-04-26T00:00:00Z",
				"2023-05-24T00:00:00Z",
				"2023-06-28T00:00:00Z",
			},
		},
		{
			name: "Kibana UI yearly default bymonth+bymonthday",
			mw: MaintWin{
				Freq:       "yearly",
				Dtstart:    "2023-03-22T00:00:00.000Z",
				Tzid:       "UTC",
				Interval:   1,
				Duration:   time.Hour,
				Bymonth:    []int{3},
				Bymonthday: []int{22},
				Count:      3,
			},
			wantISO: []string{
				"2023-03-22T00:00:00Z",
				"2024-03-22T00:00:00Z",
				"2025-03-22T00:00:00Z",
			},
		},
		{
			name: "Kibana UI custom weekly WE+TH",
			mw: MaintWin{
				Freq:      "weekly",
				Dtstart:   "2023-03-22T00:00:00.000Z",
				Tzid:      "UTC",
				Interval:  1,
				Duration:  time.Hour,
				Byweekday: []string{"WE", "TH"},
				Count:     6,
			},
			wantISO: []string{
				"2023-03-22T00:00:00Z",
				"2023-03-23T00:00:00Z",
				"2023-03-29T00:00:00Z",
				"2023-03-30T00:00:00Z",
				"2023-04-05T00:00:00Z",
				"2023-04-06T00:00:00Z",
			},
		},
		{
			name: "Kibana UI custom monthly by day 22",
			mw: MaintWin{
				Freq:       "monthly",
				Dtstart:    "2023-03-22T00:00:00.000Z",
				Tzid:       "UTC",
				Interval:   1,
				Duration:   time.Hour,
				Bymonthday: []int{22},
				Count:      4,
			},
			wantISO: []string{
				"2023-03-22T00:00:00Z",
				"2023-04-22T00:00:00Z",
				"2023-05-22T00:00:00Z",
				"2023-06-22T00:00:00Z",
			},
		},
		{
			name: "Kibana UI daily WE count 3 (ends after x)",
			mw: MaintWin{
				Freq:      "daily",
				Dtstart:   "2023-03-22T00:00:00.000Z",
				Tzid:      "UTC",
				Interval:  1,
				Duration:  time.Hour,
				Byweekday: []string{"WE"},
				Count:     3,
			},
			wantISO: []string{
				"2023-03-22T00:00:00Z",
				"2023-03-29T00:00:00Z",
				"2023-04-05T00:00:00Z",
			},
		},
		{
			name: "monthly bymonthday 31 skips short months like Kibana",
			mw: MaintWin{
				Freq:       "monthly",
				Dtstart:    "2025-01-31T00:00:00.000Z",
				Tzid:       "UTC",
				Interval:   1,
				Duration:   time.Hour,
				Bymonthday: []int{31},
				Count:      4,
			},
			wantISO: []string{
				"2025-01-31T00:00:00Z",
				"2025-03-31T00:00:00Z",
				"2025-05-31T00:00:00Z",
				"2025-07-31T00:00:00Z",
			},
		},
		{
			name: "weekly Saturday Europe/Madrid keeps local midnight across DST",
			mw: MaintWin{
				Freq:      "weekly",
				Dtstart:   "2023-01-06T23:00:00Z",
				Tzid:      "Europe/Madrid",
				Interval:  1,
				Duration:  time.Hour,
				Byweekday: []string{"SA"},
				Count:     13,
			},
			wantISO: []string{
				"2023-01-06T23:00:00Z",
				"2023-01-13T23:00:00Z",
				"2023-01-20T23:00:00Z",
				"2023-01-27T23:00:00Z",
				"2023-02-03T23:00:00Z",
				"2023-02-10T23:00:00Z",
				"2023-02-17T23:00:00Z",
				"2023-02-24T23:00:00Z",
				"2023-03-03T23:00:00Z",
				"2023-03-10T23:00:00Z",
				"2023-03-17T23:00:00Z",
				"2023-03-24T23:00:00Z",
				"2023-03-31T22:00:00Z",
			},
		},
		{
			name: "weekly Saturday Europe/Madrid keeps local midnight across fall-back",
			mw: MaintWin{
				Freq:      "weekly",
				Dtstart:   "2023-10-20T22:00:00Z",
				Tzid:      "Europe/Madrid",
				Interval:  1,
				Duration:  time.Hour,
				Byweekday: []string{"SA"},
				Count:     3,
			},
			wantISO: []string{
				"2023-10-20T22:00:00Z",
				"2023-10-27T22:00:00Z",
				"2023-11-03T23:00:00Z",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := c.mw.Parse()
			require.NoError(t, err, "Parse should succeed for Kibana-shaped rule")
			got := r.All()
			require.Len(t, got, len(c.wantISO), "occurrence count: got %v", isoTimes(got))
			assert.Equal(t, c.wantISO, isoTimes(got), "occurrences should match Kibana @kbn/rrule")
		})
	}
}

func TestKibanaIsActiveMatchesOccurrences(t *testing.T) {
	mw := MaintWin{
		Freq:      "weekly",
		Dtstart:   "2023-01-06T23:00:00Z",
		Tzid:      "Europe/Madrid",
		Interval:  1,
		Duration:  time.Hour,
		Byweekday: []string{"SA"},
	}
	r, err := mw.Parse()
	require.NoError(t, err)
	pmw := ParsedMaintWin{Rule: r, Duration: mw.Duration}

	assert.True(t, pmw.IsActive(mustTime("2023-01-06T23:00:00Z")), "window start should be active")
	assert.True(t, pmw.IsActive(mustTime("2023-01-06T23:30:00Z")), "first Saturday local occurrence should be active")
	assert.True(t, pmw.IsActive(mustTime("2023-01-07T00:00:00Z")), "Kibana lte includes exact window end")
	assert.False(t, pmw.IsActive(mustTime("2023-01-07T00:00:01Z")), "after window end should be inactive")
	assert.False(t, pmw.IsActive(mustTime("2023-01-07T23:30:00Z")), "Saturday UTC (Sunday Madrid) should not be active")
	assert.True(t, pmw.IsActive(mustTime("2023-01-13T23:30:00Z")), "next Saturday Madrid should be active")
	assert.True(t, pmw.IsActive(mustTime("2023-03-31T22:30:00Z")), "Saturday midnight CEST after DST should be active")
	assert.False(t, pmw.IsActive(mustTime("2023-03-31T23:00:01Z")), "after CEST Saturday window should be inactive")
}

func TestKibanaOneShotIsActiveOnlyForDuration(t *testing.T) {
	mw := MaintWin{
		Freq:     "yearly",
		Dtstart:  "2025-06-10T11:40:29Z",
		Tzid:     "Europe/Berlin",
		Count:    1,
		Duration: 30 * time.Minute,
	}
	r, err := mw.Parse()
	require.NoError(t, err, "Parse should succeed for Kibana one-shot yearly count 1")
	pmw := ParsedMaintWin{Rule: r, Duration: mw.Duration}

	assert.True(t, pmw.IsActive(mustTime("2025-06-10T11:40:29Z")), "one-shot start should be active")
	assert.True(t, pmw.IsActive(mustTime("2025-06-10T12:10:29Z")), "Kibana lte includes exact one-shot end")
	assert.False(t, pmw.IsActive(mustTime("2025-06-10T12:10:30Z")), "after one-shot duration should be inactive")
	assert.False(t, pmw.IsActive(mustTime("2026-06-10T11:40:29Z")), "count 1 must not recur the next year")
}

func isoTimes(ts []time.Time) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.UTC().Format(time.RFC3339)
	}
	return out
}

func mustTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}
