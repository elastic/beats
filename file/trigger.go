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
	"time"
)

// rotateReason is the reason why file rotation occurred.
type rotateReason uint32

const (
	rotateReasonNoRotate rotateReason = iota
	rotateReasonInitializing
	rotateReasonFileSize
	rotateReasonManualTrigger
	rotateReasonTimeInterval
)

func (rr rotateReason) String() string {
	switch rr {
	case rotateReasonInitializing:
		return "initializing"
	case rotateReasonFileSize:
		return "file size"
	case rotateReasonManualTrigger:
		return "manual trigger"
	case rotateReasonTimeInterval:
		return "time interval"
	default:
		return "unknown"
	}
}

// trigger interface causes the log writer to rotate the active file.
type trigger interface {
	TriggerRotation(dataLen uint) rotateReason
}

func newTriggers(rotateOnStartup bool, interval time.Duration, maxSizeBytes uint, clock clock) []trigger {
	triggers := make([]trigger, 0)

	if rotateOnStartup {
		triggers = append(triggers, &initTrigger{})
	}
	if interval > 0 {
		triggers = append(triggers, newIntervalTrigger(interval, clock))
	}
	if maxSizeBytes > 0 {
		triggers = append(triggers, &sizeTrigger{maxSizeBytes: maxSizeBytes, size: 0})
	}
	return triggers
}

// initTrigger is triggered once on startup.
type initTrigger struct {
	triggered bool
}

func (t *initTrigger) TriggerRotation(_ uint) rotateReason {
	if !t.triggered {
		t.triggered = true
		return rotateReasonInitializing
	}
	return rotateReasonNoRotate
}

// sizeTrigger starts a rotation when the file reaches the configured size.
type sizeTrigger struct {
	maxSizeBytes uint
	size         uint
}

func (t *sizeTrigger) TriggerRotation(dataLen uint) rotateReason {
	if t.size+dataLen > t.maxSizeBytes {
		t.size = 0
		return rotateReasonFileSize
	}
	t.size += dataLen
	return rotateReasonNoRotate
}

// intervalTrigger rotates the files after the configured interval.
type intervalTrigger struct {
	interval    time.Duration
	clock       clock
	lastRotate  time.Time
	newInterval func(lastTime time.Time, currentTime time.Time) bool
}

type clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

func newIntervalTrigger(interval time.Duration, clock clock) trigger {
	t := intervalTrigger{interval: interval, clock: clock}

	switch interval {
	case time.Second:
		t.newInterval = newSecond
	case time.Minute:
		t.newInterval = newMinute
	case time.Hour:
		t.newInterval = newHour
	case 24 * time.Hour: // calendar day
		t.newInterval = newDay
	case 7 * 24 * time.Hour: // calendar week
		t.newInterval = newWeek
	case 30 * 24 * time.Hour: // calendar month
		t.newInterval = newMonth
	case 365 * 24 * time.Hour: // calendar year
		t.newInterval = newYear
	default:
		t.newInterval = func(lastTime time.Time, currentTime time.Time) bool {
			lastInterval := lastTime.Unix() / (int64(t.interval) / int64(time.Second))
			currentInterval := currentTime.Unix() / (int64(t.interval) / int64(time.Second))
			return lastInterval != currentInterval
		}
	}
	return &t
}

func (t *intervalTrigger) TriggerRotation(_ uint) rotateReason {
	now := t.clock.Now()
	if t.newInterval(t.lastRotate, now) {
		t.lastRotate = now
		return rotateReasonTimeInterval
	}
	return rotateReasonNoRotate
}

func newSecond(lastTime time.Time, currentTime time.Time) bool {
	return lastTime.Second() != currentTime.Second() || newMinute(lastTime, currentTime)
}

func newMinute(lastTime time.Time, currentTime time.Time) bool {
	return lastTime.Minute() != currentTime.Minute() || newHour(lastTime, currentTime)
}

func newHour(lastTime time.Time, currentTime time.Time) bool {
	return lastTime.Hour() != currentTime.Hour() || newDay(lastTime, currentTime)
}

func newDay(lastTime time.Time, currentTime time.Time) bool {
	return lastTime.Day() != currentTime.Day() || newMonth(lastTime, currentTime)
}

func newWeek(lastTime time.Time, currentTime time.Time) bool {
	lastYear, lastWeek := lastTime.ISOWeek()
	currentYear, currentWeek := currentTime.ISOWeek()
	return lastWeek != currentWeek ||
		lastYear != currentYear
}

func newMonth(lastTime time.Time, currentTime time.Time) bool {
	return lastTime.Month() != currentTime.Month() || newYear(lastTime, currentTime)
}

func newYear(lastTime time.Time, currentTime time.Time) bool {
	return lastTime.Year() != currentTime.Year()
}
