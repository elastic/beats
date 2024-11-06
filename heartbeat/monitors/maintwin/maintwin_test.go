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
				Freq:     3,
				Dtstart:  time.Now().Format(time.RFC3339),
				Duration: mustParseDuration("2h"),
				Byweekday: []string{"SU", "MO", "TU", "WE", "TH", "FR", "SA"},
				Count:   10,
			},
			// add 30 minutes, 1 hour, 1 hour 30 minutes to the start time
			[]string{time.Now().Add(30 * time.Minute).Format(time.RFC3339), time.Now().Add(60 * time.Minute).Format(time.RFC3339), time.Now().Add(90 * time.Minute).Format(time.RFC3339)},
			[]string{time.Now().Add(180 * time.Minute).Format(time.RFC3339), time.Now().Add(540 * time.Minute).Format(time.RFC3339)},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rules := []*rrule.RRule{}
			r, err := c.mw.Parse()
			require.NoError(t, err)
			rules = append(rules, r)
			pmw := ParsedMaintWin{Rules: rules}
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
