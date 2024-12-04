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

//go:build !windows

package file

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInitTrigger(t *testing.T) {
	var trigger initTrigger
	assert.Equal(t, trigger.TriggerRotation(0), rotateReasonInitializing)
	assert.Equal(t, trigger.TriggerRotation(0), rotateReasonNoRotate)
	assert.Equal(t, trigger.TriggerRotation(0), rotateReasonNoRotate)
	assert.Equal(t, trigger.TriggerRotation(0), rotateReasonNoRotate)
}

func TestSizeTrigger(t *testing.T) {
	trigger := sizeTrigger{
		maxSizeBytes: 5,
		size:         0,
	}

	assert.EqualValues(t, trigger.size, 0)
	assert.Equal(t, trigger.TriggerRotation(1), rotateReasonNoRotate)
	assert.EqualValues(t, trigger.size, 1)
	assert.Equal(t, trigger.TriggerRotation(1), rotateReasonNoRotate)
	assert.EqualValues(t, trigger.size, 2)
	assert.Equal(t, trigger.TriggerRotation(1), rotateReasonNoRotate)
	assert.EqualValues(t, trigger.size, 3)
	assert.Equal(t, trigger.TriggerRotation(1), rotateReasonNoRotate)
	assert.EqualValues(t, trigger.size, 4)
	assert.Equal(t, trigger.TriggerRotation(1), rotateReasonNoRotate)
	assert.EqualValues(t, trigger.size, 5)
	assert.Equal(t, trigger.TriggerRotation(1), rotateReasonFileSize)
	assert.EqualValues(t, trigger.size, 0)
}

type always20240615 struct{}

func (always20240615) Now() time.Time {
	return time.Date(2024, 06, 15, 12, 30, 30, 0, time.UTC)
}

func TestIntervalTrigger(t *testing.T) {
	var ignored uint = 1

	var testCases = []struct {
		duration    string
		afterSecond bool
		afterMinute bool
		afterHour   bool
		afterDay    bool
		afterWeek   bool
		afterMonth  bool
		afterYear   bool
	}{
		{"1s", true, true, true, true, true, true, true},
		{"1m", false, true, true, true, true, true, true},
		{"1h", false, false, true, true, true, true, true},
		{"24h", false, false, false, true, true, true, true},
		{"168h", false, false, false, false, true, true, true},    // week: 7 * 24 = 168
		{"720h", false, false, false, false, false, true, true},   // month:30 * 24 = 720
		{"8760h", false, false, false, false, false, false, true}, // year: 24 * 365 = 8760
	}

	clock := &always20240615{}

	for _, testCase := range testCases {
		duration, err := time.ParseDuration(testCase.duration)
		assert.Nil(t, err)
		genericTrigger := newIntervalTrigger(duration, clock)
		trigger, ok := genericTrigger.(*intervalTrigger)
		assert.True(t, ok)

		// ensure lastRotate is initialized
		assert.NotZero(t, trigger.lastRotate)

		// Should not fire immediately
		assert.Equal(t, trigger.TriggerRotation(ignored), rotateReasonNoRotate)

		// Test after a second and ensure it doesn't fire immediately after
		trigger.lastRotate = clock.Now().Add(time.Second * -1)
		assert.Equal(t, trigger.TriggerRotation(ignored) == rotateReasonTimeInterval, testCase.afterSecond)
		assert.Equal(t, trigger.TriggerRotation(ignored), rotateReasonNoRotate)

		// Test after a minute and ensure it doesn't fire immediately after
		trigger.lastRotate = clock.Now().Add(time.Minute * -1)
		assert.Equal(t, trigger.TriggerRotation(ignored) == rotateReasonTimeInterval, testCase.afterMinute)
		assert.Equal(t, trigger.TriggerRotation(ignored), rotateReasonNoRotate)

		// Test after an hour and ensure it doesn't fire immediately after
		trigger.lastRotate = clock.Now().Add(time.Hour * -1)
		assert.Equal(t, trigger.TriggerRotation(ignored) == rotateReasonTimeInterval, testCase.afterHour)
		assert.Equal(t, trigger.TriggerRotation(ignored), rotateReasonNoRotate)

		// Test after a day and ensure it doesn't fire immediately after
		trigger.lastRotate = clock.Now().Add(time.Hour * -24)
		assert.Equal(t, trigger.TriggerRotation(ignored) == rotateReasonTimeInterval, testCase.afterDay)
		assert.Equal(t, trigger.TriggerRotation(ignored), rotateReasonNoRotate)

		// Test after a week and ensure it doesn't fire immediately after
		trigger.lastRotate = clock.Now().Add(time.Hour * -24 * 7)
		assert.Equal(t, trigger.TriggerRotation(ignored) == rotateReasonTimeInterval, testCase.afterWeek)
		assert.Equal(t, trigger.TriggerRotation(ignored), rotateReasonNoRotate)

		// Test after a month and ensure it doesn't fire immediately after
		trigger.lastRotate = clock.Now().Add(time.Hour * -24 * 31)
		assert.Equal(t, trigger.TriggerRotation(ignored) == rotateReasonTimeInterval, testCase.afterMonth)
		assert.Equal(t, trigger.TriggerRotation(ignored), rotateReasonNoRotate)

		// Test after a year and ensure it doesn't fire immediately after
		trigger.lastRotate = clock.Now().Add(time.Hour * -24 * 365)
		assert.Equal(t, trigger.TriggerRotation(ignored) == rotateReasonTimeInterval, testCase.afterYear)
		assert.Equal(t, trigger.TriggerRotation(ignored), rotateReasonNoRotate)
	}
}
