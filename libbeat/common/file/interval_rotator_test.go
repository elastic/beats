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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSecondRotator(t *testing.T) {
	a := newMockIntervalRotator(time.Second)

	clock := &testClock{time.Date(2018, 12, 31, 0, 0, 1, 100, time.Local)}
	a.clock = clock
	a.lastRotate = a.clock.Now()
	assert.Equal(t, "foo-2018-12-31-00-00-01-", a.LogPrefix("foo", time.Now()))
}

func TestMinuteRotator(t *testing.T) {
	a := newMockIntervalRotator(time.Minute)

	clock := &testClock{time.Date(2018, 12, 31, 0, 1, 1, 0, time.Local)}
	a.clock = clock
	a.lastRotate = a.clock.Now()
	assert.Equal(t, "foo-2018-12-31-00-01-", a.LogPrefix("foo", time.Now()))
}

func TestHourlyRotator(t *testing.T) {
	a := newMockIntervalRotator(time.Hour)

	clock := &testClock{time.Date(2018, 12, 31, 1, 0, 1, 0, time.Local)}
	a.clock = clock
	a.lastRotate = a.clock.Now()
	assert.Equal(t, "foo-2018-12-31-01-", a.LogPrefix("foo", time.Now()))
}

func TestDailyRotator(t *testing.T) {
	a := newMockIntervalRotator(24 * time.Hour)

	clock := &testClock{time.Date(2018, 12, 31, 0, 0, 0, 0, time.Local)}
	a.clock = clock
	a.lastRotate = a.clock.Now()
	assert.Equal(t, "foo-2018-12-31-", a.LogPrefix("foo", time.Now()))
}

func TestWeeklyRotator(t *testing.T) {
	a := newMockIntervalRotator(7 * 24 * time.Hour)

	// Monday, 2018-Dec-31
	clock := &testClock{time.Date(2018, 12, 31, 0, 0, 0, 0, time.Local)}
	a.clock = clock
	a.lastRotate = a.clock.Now()
	assert.Equal(t, "foo-2019-01-", a.LogPrefix("foo", time.Now()))

	// Monday, 2019-Jan-7
	clock.time = clock.time.Add(7 * 24 * time.Hour)
	a.lastRotate = a.clock.Now()
	assert.Equal(t, "foo-2019-02-", a.LogPrefix("foo", time.Now()))
}

func TestMonthlyRotator(t *testing.T) {
	a := newMockIntervalRotator(30 * 24 * time.Hour)

	clock := &testClock{time.Date(2018, 12, 1, 0, 0, 0, 0, time.Local)}
	a.clock = clock
	a.lastRotate = a.clock.Now()
	assert.Equal(t, "foo-2018-12-", a.LogPrefix("foo", time.Now()))

	clock.time = clock.time.Add(30 * 24 * time.Hour)
	assert.Equal(t, "foo-2018-12-", a.LogPrefix("foo", time.Now()))

	clock.time = clock.time.Add(24 * time.Hour)
	a.lastRotate = a.clock.Now()
	assert.Equal(t, "foo-2019-01-", a.LogPrefix("foo", time.Now()))
}

func TestYearlyRotator(t *testing.T) {
	a := newMockIntervalRotator(365 * 24 * time.Hour)

	clock := &testClock{time.Date(2018, 12, 31, 0, 0, 0, 0, time.Local)}
	a.clock = clock
	a.lastRotate = a.clock.Now()
	assert.Equal(t, "foo-2018-", a.LogPrefix("foo", time.Now()))

	clock.time = clock.time.Add(23 * time.Hour)
	assert.Equal(t, "foo-2018-", a.LogPrefix("foo", time.Now()))

	clock.time = clock.time.Add(time.Hour)
	a.lastRotate = a.clock.Now()
	assert.Equal(t, "foo-2019-", a.LogPrefix("foo", time.Now()))
}

func TestArbitraryIntervalRotator(t *testing.T) {
	a := newMockIntervalRotator(3 * time.Second)

	// Monday, 2018-Dec-31
	clock := &testClock{time.Date(2018, 12, 31, 0, 0, 1, 0, time.Local)}
	a.clock = clock
	assert.Equal(t, "foo-2018-12-30-00-00-00-", a.LogPrefix("foo", time.Date(2018, 12, 30, 0, 0, 0, 0, time.Local)))
	a.lastRotate = a.clock.Now()
	assert.Equal(t, "foo-2018-12-31-00-00-00-", a.LogPrefix("foo", time.Now()))

	clock.time = clock.time.Add(time.Second)
	assert.Equal(t, "foo-2018-12-31-00-00-00-", a.LogPrefix("foo", time.Now()))

	clock.time = clock.time.Add(time.Second)
	a.lastRotate = a.clock.Now()
	assert.Equal(t, "foo-2018-12-31-00-00-03-", a.LogPrefix("foo", time.Now()))

	clock.time = clock.time.Add(time.Second)
	assert.Equal(t, "foo-2018-12-31-00-00-03-", a.LogPrefix("foo", time.Now()))

	clock.time = clock.time.Add(time.Second)
	assert.Equal(t, "foo-2018-12-31-00-00-03-", a.LogPrefix("foo", time.Now()))

	clock.time = clock.time.Add(time.Second)
	a.lastRotate = a.clock.Now()
	assert.Equal(t, "foo-2018-12-31-00-00-06-", a.LogPrefix("foo", time.Now()))
}

func TestIntervalIsTruncatedToSeconds(t *testing.T) {
	a := newMockIntervalRotator(2345 * time.Millisecond)
	assert.Equal(t, 2*time.Second, a.interval)
}

type testClock struct {
	time time.Time
}

func (t testClock) Now() time.Time {
	return t.time
}

func newMockIntervalRotator(interval time.Duration) *intervalRotator {
	r := newIntervalRotator(nil, interval, "foo").(*intervalRotator)
	return r
}
