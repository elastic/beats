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
	"time"

	"github.com/teambition/rrule-go"
)

var weekdayLookup = map[string]rrule.Weekday{
	"MO": rrule.MO, "TU": rrule.TU, "WE": rrule.WE, "TH": rrule.TH, "FR": rrule.FR, "SA": rrule.SA, "SU": rrule.SU,
}

type MaintWin struct {
	Freq       rrule.Frequency `config:"freq" validate:"required"`
	Dtstart    string          `config:"dtstart" validate:"required"`
	Interval   int             `config:"interval"`
	Duration   time.Duration   `config:"duration" validate:"required"`
	Wkst       rrule.Weekday   `config:"wkst"`
	Count      int             `config:"count"`
	Bysetpos   []int           `config:"bysetpos"`
	Bymonth    []int           `config:"bymonth"`
	Bymonthday []int           `config:"bymonthday"`
	Byyearday  []int           `config:"byyearday"`
	Byweekno   []int           `config:"byweekno"`
	Byweekday  []string        `config:"byweekday"`
	Byhour     []int           `config:"byhour"`
	Byminute   []int           `config:"byminute"`
	Bysecond   []int           `config:"bysecond"`
	Byeaster   []int           `config:"byeaster"`
}

func (mw *MaintWin) Parse() (r *rrule.RRule, err error) {

	dtstart, err := time.Parse(time.RFC3339, mw.Dtstart)
	if err != nil {
		return nil, err
	}

	// Convert the string weekdays to rrule.Weekday
	weekdays := []rrule.Weekday{}
	for _, wd := range mw.Byweekday {
		weekdays = append(weekdays, weekdayLookup[wd])
	}

	dtstart = dtstart.UTC()

	count := mw.Count
	if count == 0 {
		count = 1000
	}

	r, err = rrule.NewRRule(rrule.ROption{
		Freq:       mw.Freq,
		Count:      count,
		Dtstart:    dtstart,
		Interval:   mw.Interval,
		Byweekday:  weekdays,
		Byhour:     mw.Byhour,
		Byminute:   mw.Byminute,
		Bysecond:   mw.Bysecond,
		Byeaster:   mw.Byeaster,
		Bysetpos:   mw.Bysetpos,
		Bymonth:    mw.Bymonth,
		Byweekno:   mw.Byweekno,
		Byyearday:  mw.Byyearday,
		Bymonthday: mw.Bymonthday,
		Wkst:       mw.Wkst,
	})
	if err != nil {
		return nil, err
	}

	return r, nil
}

type ParsedMaintWin struct {
	Rules     []*rrule.RRule
	Durations []time.Duration // Store durations in parallel
}

func (pmw ParsedMaintWin) IsActive(tOrig time.Time) bool {
	tOrig = tOrig.UTC()
	for i, r := range pmw.Rules {
		window := r.Before(tOrig, true)
		if tOrig.Before(window.Add(pmw.Durations[i])) {
			return true
		}
	}
	return false
}
