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

package syslog

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSeverity(t *testing.T) {
	e := newEvent()
	e.SetPriority([]byte("13"))
	assert.Equal(t, 5, e.Severity())
}

func TestFacility(t *testing.T) {
	e := newEvent()
	e.SetPriority([]byte("13"))
	assert.Equal(t, 1, e.Facility())
}

func TestHasPriority(t *testing.T) {
	e := newEvent()
	e.SetPriority([]byte("13"))
	assert.True(t, e.HasPriority())
	assert.Equal(t, 13, e.Priority())
	assert.Equal(t, 5, e.Severity())
	assert.Equal(t, 1, e.Facility())
}

func TestNoPrioritySet(t *testing.T) {
	e := newEvent()
	assert.False(t, e.HasPriority())
	assert.Equal(t, -1, e.Priority())
	assert.Equal(t, -1, e.Severity())
	assert.Equal(t, -1, e.Facility())
}

func TestHasPid(t *testing.T) {
	e := newEvent()
	assert.False(t, e.HasPid())
	e.SetPid([]byte(strconv.Itoa(20)))
	assert.True(t, e.HasPid())
}

func TestDateParsing(t *testing.T) {
	// 2018-09-12T18:14:04.537585-07:00
	e := newEvent()
	e.SetYear([]byte("2018"))
	e.SetDay(itb(12))
	e.SetMonth([]byte("Sept"))
	e.SetHour(itb(18))
	e.SetMinute(itb(14))
	e.SetSecond(itb(0o4))
	e.SetNanosecond(itb(5555))

	// Use google parser to compare.
	t1, _ := time.Parse(time.RFC3339, "2018-09-12T18:14:04.5555-07:00")
	t1 = t1.UTC()
	t2, _ := time.Parse(time.RFC3339, "2018-09-12T18:14:04.5555+07:00")
	t2 = t2.UTC()
	alreadyutc := time.Date(2018, 9, 12, 18, 14, 0o4, 555500000, time.UTC)

	tests := []struct {
		name     string
		tzBytes  []byte
		expected time.Time
	}{
		{name: "-07:00", tzBytes: []byte("-07:00"), expected: t1},
		{name: "-0700", tzBytes: []byte("-0700"), expected: t1},
		{name: "-07", tzBytes: []byte("-07"), expected: t1},
		{name: "+07:00", tzBytes: []byte("+07:00"), expected: t2},
		{name: "+0700", tzBytes: []byte("+0700"), expected: t2},
		{name: "+07", tzBytes: []byte("+07"), expected: t2},
		{name: "z+00:00", tzBytes: []byte("z+00:00"), expected: alreadyutc},
		{name: "z+0000", tzBytes: []byte("z+0000"), expected: alreadyutc},
		{name: "z+00", tzBytes: []byte("z+00"), expected: alreadyutc},
		{name: "Z+00:00", tzBytes: []byte("Z+00:00"), expected: alreadyutc},
		{name: "Z+0000", tzBytes: []byte("Z+0000"), expected: alreadyutc},
		{name: "Z+00", tzBytes: []byte("Z+00"), expected: alreadyutc},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e.SetTimeZone(test.tzBytes)
			new := e.Timestamp(nil)
			assert.Equal(t, test.expected, new)
		})
	}
}

func TestNanosecondParsing(t *testing.T) {
	e := newEvent()
	e.SetYear([]byte("2018"))
	e.SetDay(itb(12))
	e.SetMonth([]byte("Sept"))
	e.SetHour(itb(18))
	e.SetMinute(itb(14))
	e.SetSecond(itb(0o4))

	// Use google parser to compare.
	dt := func(s string) int {
		ti, _ := time.Parse(time.RFC3339, s)
		return ti.UTC().Nanosecond()
	}

	tests := []struct {
		name       string
		nanosecond []byte
		expected   int
	}{
		{nanosecond: []byte("5555"), expected: dt("2018-09-12T18:14:04.5555-07:00")},
		{nanosecond: []byte("5"), expected: dt("2018-09-12T18:14:04.5-07:00")},
		{nanosecond: []byte("0005"), expected: dt("2018-09-12T18:14:04.0005-07:00")},
		{nanosecond: []byte("000545"), expected: dt("2018-09-12T18:14:04.000545-07:00")},
		{nanosecond: []byte("00012345"), expected: dt("2018-09-12T18:14:04.00012345-07:00")},
	}

	for _, test := range tests {
		t.Run(string(test.nanosecond), func(t *testing.T) {
			e.SetNanosecond(test.nanosecond)
			assert.Equal(t, test.expected, e.Nanosecond())
		})
	}
}

func TestIsValid(t *testing.T) {
	e := newEvent()
	assert.False(t, e.IsValid())

	now := time.Now()

	e.SetDay(itb(now.Day()))
	assert.False(t, e.IsValid())

	e.SetMonth([]byte(now.Month().String()))
	assert.False(t, e.IsValid())

	e.SetHour(itb(now.Hour()))
	assert.False(t, e.IsValid())

	e.SetMinute(itb(now.Minute()))
	assert.False(t, e.IsValid())

	e.SetSecond(itb(now.Second()))
	assert.False(t, e.IsValid())

	e.SetMessage([]byte("hello world"))
	assert.True(t, e.IsValid())
}

func itb(i int) []byte {
	if i < 10 {
		return []byte(fmt.Sprintf("0%d", i))
	}
	return []byte(strconv.Itoa(i))
}
