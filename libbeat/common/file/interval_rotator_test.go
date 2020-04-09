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
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSecondRotator(t *testing.T) {
	a, err := newMockIntervalRotator(time.Second)
	if err != nil {
		t.Fatal(err)
	}

	clock := &testClock{time.Date(2018, 12, 31, 0, 0, 1, 100, time.Local)}
	a.clock = clock
	a.Rotate()
	assert.Equal(t, "foo-2018-12-31-00-00-01-", a.LogPrefix("foo", time.Now()))

	n := a.NewInterval()
	assert.False(t, n)

	clock.time = clock.time.Add(900 * time.Millisecond)
	n = a.NewInterval()
	assert.False(t, n)
	assert.Equal(t, "foo-2018-12-31-00-00-01-", a.LogPrefix("foo", time.Now()))

	clock.time = clock.time.Add(100 * time.Millisecond)
	n = a.NewInterval()
	assert.True(t, n)
	a.Rotate()
	assert.Equal(t, "foo-2018-12-31-00-00-02-", a.LogPrefix("foo", time.Now()))
}

func TestMinuteRotator(t *testing.T) {
	a, err := newMockIntervalRotator(time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	clock := &testClock{time.Date(2018, 12, 31, 0, 1, 1, 0, time.Local)}
	a.clock = clock
	a.Rotate()
	assert.Equal(t, "foo-2018-12-31-00-01-", a.LogPrefix("foo", time.Now()))

	n := a.NewInterval()
	assert.False(t, n)

	clock.time = clock.time.Add(58 * time.Second)
	n = a.NewInterval()
	assert.False(t, n)
	assert.Equal(t, "foo-2018-12-31-00-01-", a.LogPrefix("foo", time.Now()))

	clock.time = clock.time.Add(time.Second)
	n = a.NewInterval()
	assert.True(t, n)
	a.Rotate()
	assert.Equal(t, "foo-2018-12-31-00-02-", a.LogPrefix("foo", time.Now()))
}

func TestHourlyRotator(t *testing.T) {
	a, err := newMockIntervalRotator(time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	clock := &testClock{time.Date(2018, 12, 31, 1, 0, 1, 0, time.Local)}
	a.clock = clock
	a.Rotate()
	assert.Equal(t, "foo-2018-12-31-01-", a.LogPrefix("foo", time.Now()))

	n := a.NewInterval()
	assert.False(t, n)

	clock.time = clock.time.Add(58 * time.Minute)
	n = a.NewInterval()
	assert.False(t, n)
	assert.Equal(t, "foo-2018-12-31-01-", a.LogPrefix("foo", time.Now()))

	clock.time = clock.time.Add(time.Minute + 59*time.Second)
	n = a.NewInterval()
	assert.True(t, n)
	a.Rotate()
	assert.Equal(t, "foo-2018-12-31-02-", a.LogPrefix("foo", time.Now()))
}

func TestDailyRotator(t *testing.T) {
	a, err := newMockIntervalRotator(24 * time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	clock := &testClock{time.Date(2018, 12, 31, 0, 0, 0, 0, time.Local)}
	a.clock = clock
	a.Rotate()
	assert.Equal(t, "foo-2018-12-31-", a.LogPrefix("foo", time.Now()))

	n := a.NewInterval()
	assert.False(t, n)

	clock.time = clock.time.Add(23 * time.Hour)
	n = a.NewInterval()
	assert.False(t, n)
	assert.Equal(t, "foo-2018-12-31-", a.LogPrefix("foo", time.Now()))

	clock.time = clock.time.Add(time.Hour)
	n = a.NewInterval()
	assert.True(t, n)
	a.Rotate()
	assert.Equal(t, "foo-2019-01-01-", a.LogPrefix("foo", time.Now()))
}

func TestWeeklyRotator(t *testing.T) {
	a, err := newMockIntervalRotator(7 * 24 * time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	// Monday, 2018-Dec-31
	clock := &testClock{time.Date(2018, 12, 31, 0, 0, 0, 0, time.Local)}
	a.clock = clock
	a.Rotate()
	assert.Equal(t, "foo-2019-01-", a.LogPrefix("foo", time.Now()))

	n := a.NewInterval()
	assert.False(t, n)

	// Sunday, 2019-Jan-6
	clock.time = clock.time.Add(6 * 24 * time.Hour)
	n = a.NewInterval()
	assert.False(t, n)
	assert.Equal(t, "foo-2019-01-", a.LogPrefix("foo", time.Now()))

	// Monday, 2019-Jan-7
	clock.time = clock.time.Add(24 * time.Hour)
	n = a.NewInterval()
	assert.True(t, n)
	a.Rotate()
	assert.Equal(t, "foo-2019-02-", a.LogPrefix("foo", time.Now()))
}

func TestMonthlyRotator(t *testing.T) {
	a, err := newMockIntervalRotator(30 * 24 * time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	clock := &testClock{time.Date(2018, 12, 1, 0, 0, 0, 0, time.Local)}
	a.clock = clock
	a.Rotate()
	assert.Equal(t, "foo-2018-12-", a.LogPrefix("foo", time.Now()))

	n := a.NewInterval()
	assert.False(t, n)

	clock.time = clock.time.Add(30 * 24 * time.Hour)
	n = a.NewInterval()
	assert.False(t, n)
	assert.Equal(t, "foo-2018-12-", a.LogPrefix("foo", time.Now()))

	clock.time = clock.time.Add(24 * time.Hour)
	n = a.NewInterval()
	assert.True(t, n)
	a.Rotate()
	assert.Equal(t, "foo-2019-01-", a.LogPrefix("foo", time.Now()))
}

func TestYearlyRotator(t *testing.T) {
	a, err := newMockIntervalRotator(365 * 24 * time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	clock := &testClock{time.Date(2018, 12, 31, 0, 0, 0, 0, time.Local)}
	a.clock = clock
	a.Rotate()
	assert.Equal(t, "foo-2018-", a.LogPrefix("foo", time.Now()))

	n := a.NewInterval()
	assert.False(t, n)

	clock.time = clock.time.Add(23 * time.Hour)
	n = a.NewInterval()
	assert.False(t, n)
	assert.Equal(t, "foo-2018-", a.LogPrefix("foo", time.Now()))

	clock.time = clock.time.Add(time.Hour)
	n = a.NewInterval()
	assert.True(t, n)
	a.Rotate()
	assert.Equal(t, "foo-2019-", a.LogPrefix("foo", time.Now()))
}

func TestArbitraryIntervalRotator(t *testing.T) {
	a, err := newMockIntervalRotator(3 * time.Second)
	if err != nil {
		t.Fatal(err)
	}

	// Monday, 2018-Dec-31
	clock := &testClock{time.Date(2018, 12, 31, 0, 0, 1, 0, time.Local)}
	a.clock = clock
	assert.Equal(t, "foo-2018-12-30-00-00-00-", a.LogPrefix("foo", time.Date(2018, 12, 30, 0, 0, 0, 0, time.Local)))
	a.Rotate()
	n := a.NewInterval()
	assert.False(t, n)
	assert.Equal(t, "foo-2018-12-31-00-00-00-", a.LogPrefix("foo", time.Now()))

	clock.time = clock.time.Add(time.Second)
	n = a.NewInterval()
	assert.False(t, n)
	assert.Equal(t, "foo-2018-12-31-00-00-00-", a.LogPrefix("foo", time.Now()))

	clock.time = clock.time.Add(time.Second)
	n = a.NewInterval()
	assert.True(t, n)
	a.Rotate()
	assert.Equal(t, "foo-2018-12-31-00-00-03-", a.LogPrefix("foo", time.Now()))

	clock.time = clock.time.Add(time.Second)
	n = a.NewInterval()
	assert.False(t, n)
	assert.Equal(t, "foo-2018-12-31-00-00-03-", a.LogPrefix("foo", time.Now()))

	clock.time = clock.time.Add(time.Second)
	n = a.NewInterval()
	assert.False(t, n)
	assert.Equal(t, "foo-2018-12-31-00-00-03-", a.LogPrefix("foo", time.Now()))

	clock.time = clock.time.Add(time.Second)
	n = a.NewInterval()
	assert.True(t, n)
	a.Rotate()
	assert.Equal(t, "foo-2018-12-31-00-00-06-", a.LogPrefix("foo", time.Now()))
}

func TestIntervalIsTruncatedToSeconds(t *testing.T) {
	a, err := newMockIntervalRotator(2345 * time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2*time.Second, a.interval)
}

func TestZeroIntervalIsNil(t *testing.T) {
	a, err := newMockIntervalRotator(0)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, a == nil)
}

func TestSelectingLastRotateTime(t *testing.T) {
	const oldestTsSec = 1586434581
	cases := map[string]struct {
		logfiles           []os.FileInfo
		expectedRotateTime time.Time
	}{
		"one file unrotated file": {
			logfiles: []os.FileInfo{
				fileInf{
					name:    "foo",
					modTime: time.Unix(oldestTsSec, 0),
				},
			},
			expectedRotateTime: time.Unix(oldestTsSec, 0),
		},
		"one file unrotated file, several rotated files": {
			logfiles: []os.FileInfo{
				fileInf{
					name:    "foo",
					modTime: time.Unix(oldestTsSec+4*60, 0),
				},
				fileInf{
					name:    "foo-01",
					modTime: time.Unix(oldestTsSec+3*60, 0),
				},
				fileInf{
					name:    "foo-02",
					modTime: time.Unix(oldestTsSec+2*60, 0),
				},
				fileInf{
					name:    "foo-03",
					modTime: time.Unix(oldestTsSec+1*60, 0),
				},
			},
			expectedRotateTime: time.Unix(oldestTsSec+4*60, 0),
		},
		"several rotated files": {
			logfiles: []os.FileInfo{
				fileInf{
					name:    "foo-01",
					modTime: time.Unix(oldestTsSec+3*60, 0),
				},
				fileInf{
					name:    "foo-02",
					modTime: time.Unix(oldestTsSec+2*60, 0),
				},
				fileInf{
					name:    "foo-03",
					modTime: time.Unix(oldestTsSec+1*60, 0),
				},
			},
			expectedRotateTime: time.Unix(oldestTsSec+3*60, 0),
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			rotatedTime := determineTimeOfLastRotation(nil, "foo", test.logfiles)
			assert.Equal(t, rotatedTime.Sub(test.expectedRotateTime), time.Duration(0))
		})
	}
}

type fileInf struct {
	name    string
	modTime time.Time
}

func (f fileInf) Name() string       { return f.name }
func (f fileInf) ModTime() time.Time { return f.modTime }
func (f fileInf) Size() int64        { return 0 }
func (f fileInf) Mode() os.FileMode  { return 0666 }
func (f fileInf) IsDir() bool        { return false }
func (f fileInf) Sys() interface{}   { return nil }

type testClock struct {
	time time.Time
}

func (t testClock) Now() time.Time {
	return t.time
}

func newMockIntervalRotator(interval time.Duration) (*intervalRotator, error) {
	return newIntervalRotator(nil, interval, false, "foo")
}
