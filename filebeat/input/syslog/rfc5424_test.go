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
	"github.com/stretchr/testify/assert"
	"testing"
)

const VersionTestTemplate = `<34>%d 2003-10-11T22:14:15.003Z mymachine.example.com su - ID47 - BOM'su root' failed for lonvick on /dev/pts/8`
const PriorityTestTemplate = `<%d>1 2003-10-11T22:14:15.003Z mymachine.example.com su - ID47 - BOM'su root' failed for lonvick on /dev/pts/8`
const TimeTestTemplate = `<22>11 2003-10-11T22:14:15.003Z mymachine.example.com su - ID47 - BOM'su root' failed for lonvick on /dev/pts/8`

type testRule struct {
	title  string
	log    []byte
	syslog event
}

func setRightTime(e *event) {
	e.year = 2003
	e.month = 10
	e.day = 11
	e.hour = 22
	e.minute = 14
	e.second = 15
	e.second = 15
	e.nanosecond = 3000000
}

func createVersionTestRule(v int, success bool) testRule {

	var rule = testRule{
		title:  fmt.Sprintf("versionTest v:%d", v),
		log:    []byte(fmt.Sprintf(VersionTestTemplate, v)),
		syslog: *newEvent(),
	}
	rule.syslog.SetPriority([]byte("34"))

	if !success {
		return rule
	}

	setRightTime(&rule.syslog)
	rule.syslog.version = v
	return rule
}

func createPriorityTestRule(v int, success bool) testRule {
	var rule = testRule{
		title:  fmt.Sprintf("priorityTest v:%d", v),
		log:    []byte(fmt.Sprintf(PriorityTestTemplate, v)),
		syslog: *newEvent(),
	}
	if !success {
		return rule
	}

	setRightTime(&rule.syslog)
	rule.syslog.priority = v
	rule.syslog.SetVersion([]byte("1"))
	return rule
}

func createTimeTestRule() testRule {
	var rule = testRule{
		title: fmt.Sprintf("TimestampTest v"),
		log:   []byte(fmt.Sprintf(TimeTestTemplate)),
		syslog: event{
			version:  11,
			priority: 22,
		},
	}
	setRightTime(&rule.syslog)
	return rule
}

func TestParseRfc5424Syslog(t *testing.T) {
	var tests []testRule

	// add some priorityTest
	tests = append(tests, createPriorityTestRule(0, true))
	tests = append(tests, createPriorityTestRule(20, true))
	tests = append(tests, createPriorityTestRule(180, true))
	tests = append(tests, createPriorityTestRule(190, true))
	tests = append(tests, createPriorityTestRule(191, true))
	tests = append(tests, createPriorityTestRule(192, false))
	tests = append(tests, createPriorityTestRule(200, false))
	tests = append(tests, createPriorityTestRule(1000, false))

	// add some version test
	tests = append(tests, createVersionTestRule(0, false))
	tests = append(tests, createVersionTestRule(30, true))
	tests = append(tests, createVersionTestRule(100, true))
	tests = append(tests, createVersionTestRule(1000, false))

	tests = append(tests, createTimeTestRule())

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s:%s", test.title, string(test.log)), func(t *testing.T) {
			l := newEvent()
			Parse5424(test.log, l)
			AssertEvent(t, test.syslog, l)
		})
	}
}

func AssertEvent(t *testing.T, except event, actual *event) {
	assert.Equal(t, except.Priority(), actual.Priority())
	assert.Equal(t, except.Version(), actual.Version())
	assert.Equal(t, except.Year(), actual.Year())
	assert.Equal(t, except.Month(), actual.Month())
	assert.Equal(t, except.Day(), actual.Day())
	assert.Equal(t, except.Hour(), actual.Hour())
	assert.Equal(t, except.Minute(), actual.Minute())
	assert.Equal(t, except.Second(), actual.Second())
	assert.Equal(t, except.Nanosecond(), actual.Nanosecond())
}
