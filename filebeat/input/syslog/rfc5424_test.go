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
	"time"
)

const BOM = "\xEF\xBB\xBF"

const VersionTestTemplate = `<34>%d 2003-10-11T22:14:15.003Z mymachine.example.com su - ID47 - ` + BOM + `'su root' failed for lonvick on /dev/pts/8`
const PriorityTestTemplate = `<%d>1 2003-10-11T22:14:15.003Z mymachine.example.com su - ID47 - ` + BOM + `'su root' failed for lonvick on /dev/pts/8`

const RfcDoc65Example1 = `<34>1 2003-10-11T22:14:15.003Z mymachine.example.com su - ID47 - ` + BOM + `'su root' failed for lonvick on /dev/pts/8`
const RfcDoc65Example2 = `<165>1 2003-08-24T05:14:15.000003-07:00 192.0.2.1 myproc 8710 - - %% It's time to make the do-nuts.`

//   Example 3 - with STRUCTURED-DATA
const RfcDoc65Example3 = `<165>1 2003-10-11T22:14:15.003Z mymachine.example.com evntslog - ID47 [exampleSDID@32473 iut="3" eventSource="Application" eventID="1011"] ` + BOM + `An application event log entry...`
const RfcDoc65Example4 = `<165>1 2003-10-11T22:14:15.003Z mymachine.example.com evntslog - ID47 [exampleSDID@32473 iut="3" eventSource="Application" eventID="1011"][examplePriority@32473 class="high"]`

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
	e.nanosecond = 3000000
}

func createVersionTestRule(v int, success bool) testRule {

	var rule = testRule{
		title: fmt.Sprintf("versionTest v:%d", v),
		log:   []byte(fmt.Sprintf(VersionTestTemplate, v)),
		syslog: event{
			priority:   34,
			version:    v,
			hostname:   "mymachine.example.com",
			appName:    "su",
			processID:  "-",
			msgID:      "ID47",
			year:       2003,
			month:      10,
			day:        11,
			hour:       22,
			minute:     14,
			second:     15,
			nanosecond: 3000000,
			message:    "'su root' failed for lonvick on /dev/pts/8",
		}}

	if !success {
		rule.syslog = *newEvent()
		rule.syslog.priority = 34
		return rule
	}

	return rule
}

func createPriorityTestRule(v int, success bool) testRule {
	var rule = testRule{
		title: fmt.Sprintf("priorityTest v:%d", v),
		log:   []byte(fmt.Sprintf(PriorityTestTemplate, v)),
		syslog: event{
			priority:   v,
			version:    1,
			hostname:   "mymachine.example.com",
			appName:    "su",
			processID:  "-",
			msgID:      "ID47",
			year:       2003,
			month:      10,
			day:        11,
			hour:       22,
			minute:     14,
			second:     15,
			message:    "'su root' failed for lonvick on /dev/pts/8",
			nanosecond: 3000000,
		},
	}
	if !success {
		rule.syslog = *newEvent()
		return rule
	}
	return rule
}

func TestRfc5424ParseHeader(t *testing.T) {
	var tests = []testRule{{
		title: fmt.Sprintf("TestHeader RfcDoc65Example1"),
		log:   []byte(fmt.Sprintf(RfcDoc65Example1)),
		syslog: event{
			priority:   34,
			version:    1,
			hostname:   "mymachine.example.com",
			appName:    "su",
			processID:  "-",
			msgID:      "ID47",
			year:       2003,
			month:      10,
			day:        11,
			hour:       22,
			minute:     14,
			second:     15,
			nanosecond: 3000000,
			message:    "'su root' failed for lonvick on /dev/pts/8",
		},
	}, {
		title: fmt.Sprintf("TestHeader RfcDoc65Example2"),
		log:   []byte(RfcDoc65Example2),
		syslog: event{
			priority:   165,
			version:    1,
			hostname:   "192.0.2.1",
			appName:    "myproc",
			processID:  "8710",
			msgID:      "-",
			year:       2003,
			month:      8,
			day:        24,
			hour:       5,
			minute:     14,
			second:     15,
			nanosecond: 3000,
			message:    `%% It's time to make the do-nuts.`,
			loc:        time.FixedZone("", -7*3600),
		},
	}}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s:%s", test.title, string(test.log)), func(t *testing.T) {
			l := newEvent()
			ParserRFC5424(test.log, l)
			AssertEvent(t, test.syslog, l)
		})
	}
}

func TestRfc5424ParseStructuredData(t *testing.T) {
	var tests = []testRule{{
		title: fmt.Sprintf("TestHeader RfcDoc65Example3"),
		log:   []byte(RfcDoc65Example3),
		syslog: event{
			priority:   165,
			version:    1,
			hostname:   "mymachine.example.com",
			appName:    "evntslog",
			processID:  "-",
			msgID:      "ID47",
			year:       2003,
			month:      10,
			day:        11,
			hour:       22,
			minute:     14,
			second:     15,
			nanosecond: 3000000,
			message:    "An application event log entry...",
			data: map[string]map[string]string{
				"exampleSDID@32473": {
					"iut":         "3",
					"eventID":     "1011",
					"eventSource": "Application",
				},
			},
		},
	}, {
		title: fmt.Sprintf("TestHeader RfcDoc65Example4"),
		log:   []byte(fmt.Sprintf(RfcDoc65Example4)),
		syslog: event{
			priority:   165,
			version:    1,
			hostname:   "mymachine.example.com",
			appName:    "evntslog",
			processID:  "-",
			msgID:      "ID47",
			year:       2003,
			month:      10,
			day:        11,
			hour:       22,
			minute:     14,
			second:     15,
			nanosecond: 3000000,
			data: map[string]map[string]string{
				"exampleSDID@32473": {
					"iut":         "3",
					"eventID":     "1011",
					"eventSource": "Application",
				},
				"examplePriority@32473": {
					"class": "high",
				},
			},
		},
	},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s:%s", test.title, string(test.log)), func(t *testing.T) {
			l := newEvent()
			ParserRFC5424(test.log, l)
			AssertEvent(t, test.syslog, l)
		})
	}
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

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s:%s", test.title, string(test.log)), func(t *testing.T) {
			l := newEvent()
			ParserRFC5424(test.log, l)
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
	assert.Equal(t, except.loc, actual.loc)
	assert.Equal(t, except.Hostname(), actual.Hostname())
	assert.Equal(t, except.AppName(), actual.AppName())
	assert.Equal(t, except.ProcID(), actual.ProcID())
	assert.Equal(t, except.MsgID(), actual.MsgID())
	assert.Equal(t, except.data, actual.data)
	assert.Equal(t, except.Message(), actual.Message())
}
