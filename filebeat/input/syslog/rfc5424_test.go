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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const BOM = "\xEF\xBB\xBF"

const (
	VersionTestTemplate  = `<34>%d 2003-10-11T22:14:15.003Z mymachine.example.com su - ID47 - ` + BOM + `'su root' failed for lonvick on /dev/pts/8`
	PriorityTestTemplate = `<%d>1 2003-10-11T22:14:15.003Z mymachine.example.com su - ID47 - ` + BOM + `'su root' failed for lonvick on /dev/pts/8`
)

// https://tools.ietf.org/html/rfc5424#section-6.5
const RfcDoc65Example1 = `<34>1 2003-10-11T22:14:15.003Z mymachine.example.com su - ID47 - ` + BOM + `'su root' failed for lonvick on /dev/pts/8`

const (
	RfcDoc65Example2          = `<165>1 2003-08-24T05:14:15.000003-07:00 192.0.2.1 myproc 8710 - - %% It's time to make the do-nuts.`
	RfcDoc65Example3          = `<165>1 2003-10-11T22:14:15.003Z mymachine.example.com evntslog - ID47 [exampleSDID@32473 iut="3" eventSource="Application" eventID="1011"] ` + BOM + `An application event log entry...`
	RfcDoc65Example4          = `<165>1 2003-10-11T22:14:15.003Z mymachine.example.com evntslog - ID47 [exampleSDID@32473 iut="3" eventSource="Application" eventID="1011"][examplePriority@32473 class="high"]`
	RfcDoc65Example4WithoutSD = `<165>1 2003-10-11T22:14:15.003Z mymachine.example.com evntslog - ID47 `
	MESSAGE                   = `An application event log entry...`
)

func getTestEvent() event {
	return event{
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
	}
}

type testRule struct {
	title    string
	log      []byte
	syslog   event
	isFailed bool
}

func runTests(rules []testRule, t *testing.T) {
	for _, rule := range rules {
		t.Run(fmt.Sprintf("%s:%s", rule.title, string(rule.log)), func(t *testing.T) {
			l := newEvent()
			ParserRFC5424(rule.log, l)
			if rule.isFailed {
				assert.Equal(t, false, l.IsValid())
				return
			} else {
				assert.Equal(t, true, l.IsValid())
			}
			AssertEvent(t, rule.syslog, l)
		})
	}
}

func TestRfc5424ParseHeader(t *testing.T) {
	tests := []testRule{{
		title:  "RfcDoc 6.5 Example1",
		log:    []byte(RfcDoc65Example1),
		syslog: getTestEvent(),
	}, {
		title: "RfcDoc 6.5 Example2",
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
	runTests(tests, t)
}

func CreateStructuredDataWithMsg(msg string, data EventData) event {
	return event{
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
		message:    msg,
		data:       data,
	}
}

func CreateStructuredData(data EventData) event {
	return CreateStructuredDataWithMsg(MESSAGE, data)
}

func CreateTest(title string, log string, syslog event) testRule {
	return testRule{
		title:    title,
		log:      []byte(log),
		syslog:   syslog,
		isFailed: false,
	}
}

func CreateParseFailTest(title string, log string, syslog event) testRule {
	return testRule{
		title:    title,
		log:      []byte(log),
		syslog:   syslog,
		isFailed: true,
	}
}

func TestRfc5424ParseStructuredData(t *testing.T) {
	tests := []testRule{
		CreateTest("RfcDoc65Example3",
			RfcDoc65Example3,
			CreateStructuredData(EventData{
				"exampleSDID@32473": {
					"iut":         "3",
					"eventID":     "1011",
					"eventSource": "Application",
				},
			})),
		CreateTest("RfcDoc65Example4",
			RfcDoc65Example4,
			CreateStructuredDataWithMsg("", EventData{
				"exampleSDID@32473": {
					"iut":         "3",
					"eventID":     "1011",
					"eventSource": "Application",
				},
				"examplePriority@32473": {
					"class": "high",
				},
			})),
		CreateTest("test structured data param value with escape",
			`<165>1 2003-10-11T22:14:15.003Z mymachine.example.com evntslog - ID47 [exampleSDID@32473 iut="\]3" eventSource="\"Application\"" eventID="1011"] `+MESSAGE,
			CreateStructuredData(EventData{
				"exampleSDID@32473": {
					"iut":         "]3",
					"eventID":     "1011",
					"eventSource": `"Application"`,
				},
			})),
		// https://tools.ietf.org/html/rfc5424#section-6.3.5
		CreateTest("RfcDoc635Example1",
			RfcDoc65Example4WithoutSD+`[exampleSDID@32473 iut="3" eventSource="Application" eventID="1011"]`,
			CreateStructuredDataWithMsg("", EventData{
				"exampleSDID@32473": {
					"iut":         "3",
					"eventID":     "1011",
					"eventSource": "Application",
				},
			})),

		CreateTest("RfcDoc635Example2",
			RfcDoc65Example4WithoutSD+`[exampleSDID@32473 iut="3" eventSource="Application" eventID="1011"][examplePriority@32473 class="high"]`,
			CreateStructuredDataWithMsg("", EventData{
				"exampleSDID@32473": {
					"iut":         "3",
					"eventID":     "1011",
					"eventSource": "Application",
				},
				"examplePriority@32473": {
					"class": "high",
				},
			})),
		CreateTest("RfcDoc635Example3",
			RfcDoc65Example4WithoutSD+`[exampleSDID@32473 iut="3" eventSource="Application" eventID="1011"] [examplePriority@32473 class="high"] `+MESSAGE,
			CreateStructuredDataWithMsg(`[examplePriority@32473 class="high"] `+MESSAGE, EventData{
				"exampleSDID@32473": {
					"iut":         "3",
					"eventID":     "1011",
					"eventSource": "Application",
				},
			})),
		CreateParseFailTest("RfcDoc635Example4",
			RfcDoc65Example4WithoutSD+`[ exampleSDID@32473 iut="3" eventSource="Application" eventID="1011" ][examplePriority@32473 class="high"]`+MESSAGE,
			CreateStructuredDataWithMsg(``, EventData{})),

		CreateTest("RfcDoc635Example5",
			RfcDoc65Example4WithoutSD+`[sigSig ver="1" rsID="1234" iut="3" signature="..."] `+MESSAGE,
			CreateStructuredDataWithMsg(MESSAGE, EventData{
				"sigSig": {
					"ver":       "1",
					"rsID":      "1234",
					"iut":       "3",
					"signature": "...",
				},
			})),
	}

	runTests(tests, t)
}

func createVersionTestRule(v int, success bool) testRule {
	rule := testRule{
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
		},
	}

	if !success {
		rule.isFailed = true
		return rule
	}

	return rule
}

func createPriorityTestRule(v int, success bool) testRule {
	rule := testRule{
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
		rule.isFailed = true
		return rule
	}
	return rule
}

func TestRfc5424SyslogParserValueBoundary(t *testing.T) {
	var tests []testRule

	// add priorityTest, 0 <= priority <= 191.
	tests = append(tests, createPriorityTestRule(0, true))
	tests = append(tests, createPriorityTestRule(20, true))
	tests = append(tests, createPriorityTestRule(180, true))
	tests = append(tests, createPriorityTestRule(190, true))
	tests = append(tests, createPriorityTestRule(191, true))
	tests = append(tests, createPriorityTestRule(192, false))
	tests = append(tests, createPriorityTestRule(200, false))
	tests = append(tests, createPriorityTestRule(1000, false))

	// add version test, version <= 999
	tests = append(tests, createVersionTestRule(0, false))
	tests = append(tests, createVersionTestRule(30, true))
	tests = append(tests, createVersionTestRule(100, true))
	tests = append(tests, createVersionTestRule(1000, false))

	runTests(tests, t)
}

func TestRfc5424SyslogParserDate(t *testing.T) {
	test := []testRule{
		{
			title: "Test two-digit mdays",
			log:   []byte(`<165>1 2003-08-07T05:14:15.000003-07:00 192.0.2.1 myproc 8710 - - %% It's time to make the do-nuts.`),
			syslog: event{
				priority:   165,
				version:    1,
				hostname:   "192.0.2.1",
				appName:    "myproc",
				processID:  "8710",
				msgID:      "-",
				year:       2003,
				month:      8,
				day:        7,
				hour:       5,
				minute:     14,
				second:     15,
				nanosecond: 3000,
				message:    `%% It's time to make the do-nuts.`,
				loc:        time.FixedZone("", -7*3600),
			},
		},
	}

	runTests(test, t)
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
