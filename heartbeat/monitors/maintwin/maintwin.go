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
	Freq       int           `config:"freq" validate:"required"`
	Dtstart    string        `config:"dtstart" validate:"required"`
	Interval   int           `config:"interval" validate:"required"`
	Duration   time.Duration `config:"duration" validate:"required"`
	Wkst       rrule.Weekday `config:"wkst"`
	Count      int           `config:"count"`
	Bysetpos   []int         `config:"bysetpos"`
	Bymonth    []int         `config:"bymonth"`
	Bymonthday []int         `config:"bymonthday"`
	Byyearday  []int         `config:"byyearday"`
	Byweekno   []int         `config:"byweekno"`
	Byweekday  []string      `config:"byweekday"`
	Byhour     []int         `config:"byhour"`
	Byminute   []int         `config:"byminute"`
	Bysecond   []int         `config:"bysecond"`
	Byeaster   []int         `config:"byeaster"`
}

func (mw *MaintWin) Parse() (r *rrule.RRule, err error) {

	dtstart, _ := time.Parse(time.RFC3339, mw.Dtstart)

	// Convert the string weekdays to rrule.Weekday
	weekdays := []rrule.Weekday{}
	for _, wd := range mw.Byweekday {
		weekdays = append(weekdays, weekdayLookup[wd])
	}

	r, _ = rrule.NewRRule(rrule.ROption{
		Freq:       rrule.Frequency(mw.Freq),
		Count:      mw.Count,
		Dtstart:    dtstart,
		Interval:   mw.Interval,
		Until:      dtstart.Add(mw.Duration),
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

	return r, nil
}

type ParsedMaintWin struct {
	Rules []*rrule.RRule
}

func (pmw ParsedMaintWin) IsActive(tOrig time.Time) bool {
	matched := false
	for _, r := range pmw.Rules {
		nextOccurance := r.After(tOrig, true)

		if tOrig.Equal(nextOccurance) || tOrig.After(nextOccurance) && tOrig.Before(r.GetUntil()) {
			matched = true
			break
		}

	}

	return matched
}
