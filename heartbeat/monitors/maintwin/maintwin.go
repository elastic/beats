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
	"strings"
	"time"

	"github.com/teambition/rrule-go"
)

var weekdayLookup = map[string]rrule.Weekday{
	"MO": rrule.MO, "TU": rrule.TU, "WE": rrule.WE, "TH": rrule.TH, "FR": rrule.FR, "SA": rrule.SA, "SU": rrule.SU,
}

type MaintWin struct {
	Freq       string        `config:"freq" validate:"required"`
	Dtstart    string        `config:"dtstart" validate:"required"`
	Interval   int           `config:"interval"`
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

func (mw *MaintWin) Parse(validateDtStart bool) (r *rrule.RRule, err error) {

	// validate the frequency, we don't support less than daily
	freq, err := rrule.StrToFreq(strings.ToUpper(mw.Freq))
	if err != nil || freq > rrule.DAILY {
		return nil, fmt.Errorf("Invalid frequency %s: only yearly, monthly, weekly, and daily are supported", mw.Freq)
	}

	dtstart, err := time.Parse(time.RFC3339, mw.Dtstart)
	if err != nil {
		return nil, err
	}

	// validate DTSTART and make sure it's not older than 2 years
	if dtstart.Before(time.Now().AddDate(-2, 0, 0)) && validateDtStart {
		return nil, fmt.Errorf(
			"invalid dtstart: %s is more than 2 years in the past. "+
				"To prevent excessive iterations, please use a more recent date.",
			dtstart.Format(time.RFC3339),
		)
	}

	// Convert the string weekdays to rrule.Weekday
	weekdays := []rrule.Weekday{}
	for _, wd := range mw.Byweekday {
		if weekday, exists := weekdayLookup[wd]; exists {
			weekdays = append(weekdays, weekday)
		}
	}

	dtstart = dtstart.UTC()

	r, err = rrule.NewRRule(rrule.ROption{
		Freq:       freq,
		Count:      mw.Count,
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
	Rule     *rrule.RRule
	Duration time.Duration
}

func (pmw ParsedMaintWin) IsActive(tOrig time.Time) bool {
	if pmw.Rule == nil {
		return false
	}
	tOrig = tOrig.UTC()
	window := pmw.Rule.Before(tOrig, true)
	return !window.IsZero() && tOrig.Before(window.Add(pmw.Duration))
}
