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

package file

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"
)

type intervalRotator struct {
	interval    time.Duration
	lastRotate  time.Time
	fileFormat  string
	clock       clock
	weekly      bool
	yearly      bool
	arbitrary   bool
	newInterval func(lastTime time.Time, currentTime time.Time, step time.Duration) bool
}

type clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

func newIntervalRotator(interval time.Duration) (*intervalRotator, error) {
	if interval == 0 {
		return nil, nil
	}
	if interval < time.Second && interval != 0 {
		return nil, errors.New("the minimum time interval for log rotation is 1 second")
	}

	ir := &intervalRotator{interval: (interval / time.Second) * time.Second} // drop fractional seconds
	ir.initialize()
	return ir, nil
}

func (r *intervalRotator) initialize() error {
	r.clock = realClock{}

	r.newInterval = newStep
	if r.interval < time.Minute {
		r.fileFormat = "2006-01-02-15-04-05"
	} else if r.interval < time.Hour {
		r.fileFormat = "2006-01-02-15-04"
	} else if r.interval < 24*time.Hour {
		r.fileFormat = "2006-01-02-15"
	} else if r.interval < 7*24*time.Hour {
		r.fileFormat = "2006-01-02"
	} else if r.interval < 30*24*time.Hour {
		r.fileFormat = ""
		r.weekly = true
	} else if r.interval < 365*24*time.Hour {
		r.fileFormat = "2006-01"
	} else if r.interval >= 365*24*time.Hour {
		r.fileFormat = "2006-01"
		r.yearly = true
	} else {
		r.arbitrary = true
		r.fileFormat = "2006-01-02-15-04-05"
		r.newInterval = func(lastTime time.Time, currentTime time.Time, step time.Duration) bool {
			lastInterval := lastTime.Unix() / (int64(r.interval) / int64(time.Second))
			currentInterval := currentTime.Unix() / (int64(r.interval) / int64(time.Second))
			return lastInterval != currentInterval
		}

	}

	return nil
}

func (r *intervalRotator) LogPrefix(filename string, modTime time.Time) string {
	var t time.Time

	t = modTime

	if r.weekly {
		y, w := t.ISOWeek()
		return fmt.Sprintf("%s-%04d-%02d-", filename, y, w)
	}
	if r.yearly {
		y, _ := t.ISOWeek()
		return fmt.Sprintf("%s-%04d-", filename, y)
	}
	if r.arbitrary {
		intervalNumber := t.Unix() / (int64(r.interval) / int64(time.Second))
		intervalStart := time.Unix(0, intervalNumber*int64(r.interval))
		return fmt.Sprintf("%s-%s-", filename, intervalStart.Format(r.fileFormat))
	}
	return fmt.Sprintf("%s-%s-", filename, t.Format(r.fileFormat))
}

func (r *intervalRotator) NewInterval() bool {
	now := r.clock.Now()
	newInterval := r.newInterval(r.lastRotate, now, r.interval)
	return newInterval
}

func (r *intervalRotator) Rotate() {
	r.lastRotate = r.clock.Now()
}

func (r *intervalRotator) SortIntervalLogs(strings []string) {
	sort.Slice(
		strings,
		func(i, j int) bool {
			return OrderIntervalLogs(strings[i]) < OrderIntervalLogs(strings[j])
		},
	)
}

// OrderIntervalLogs, when given a log filename in the form [prefix]-[formattedDate]-n
// returns the filename after zero-padding the trailing n so that foo-[date]-2 sorts
// before foo-[date]-10.
func OrderIntervalLogs(filename string) string {
	index, i, err := IntervalLogIndex(filename)
	if err == nil {
		return filename[:i] + fmt.Sprintf("%020d", index)
	}

	return ""
}

// IntervalLogIndex returns n as int given a log filename in the form [prefix]-[formattedDate]-n
func IntervalLogIndex(filename string) (uint64, int, error) {
	i := len(filename) - 1
	for ; i >= 0; i-- {
		if '0' > filename[i] || filename[i] > '9' {
			break
		}
	}
	i++

	s64 := filename[i:]
	u64, err := strconv.ParseUint(s64, 10, 64)
	return u64, i, err
}

func newStep(lastTime time.Time, currentTime time.Time, step time.Duration) bool {
	return !currentTime.Before(lastTime.Add(step))
}
