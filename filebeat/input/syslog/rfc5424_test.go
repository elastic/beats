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

type testRule struct {
	title  string
	log    []byte
	syslog event
}

func createVersionTestRule(v int, success bool) testRule {
	var except = -1
	if success {
		except = v
	}

	return testRule{
		title: fmt.Sprintf("versionTest v:%d", v),
		log:   []byte(fmt.Sprintf(VersionTestTemplate, v)),
		syslog: event{
			version:  except,
			priority: 34,
		},
	}
}

func createPriorityTestRule(v int, success bool) testRule {
	var rule = testRule{
		title: fmt.Sprintf("priorityTest v:%d", v),
		log:   []byte(fmt.Sprintf(PriorityTestTemplate, v)),
		syslog: event{
			version:  1,
			priority: v,
		},
	}
	if !success {
		rule.syslog = event{
			priority: -1,
			version: -1,
		}
	}

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

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s:%s", test.title, string(test.log)), func(t *testing.T) {
			l := newEvent()
			Parse5424(test.log, l)
			//assert.Equal(t, test.syslog.Message(), l.Message())
			//assert.Equal(t, test.syslog.Hostname(), l.Hostname())
			assert.Equal(t, test.syslog.Priority(), l.Priority())
			assert.Equal(t, test.syslog.Version(), l.Version())
			//assert.Equal(t, test.syslog.Pid(), l.Pid())
			//assert.Equal(t, test.syslog.Program(), l.Program())
			//assert.Equal(t, test.syslog.Month(), l.Month())
			//assert.Equal(t, test.syslog.Day(), l.Day())
			//assert.Equal(t, test.syslog.Hour(), l.Hour())
			//assert.Equal(t, test.syslog.Minute(), l.Minute())
			//assert.Equal(t, test.syslog.Second(), l.Second())
			//assert.Equal(t, test.syslog.Nanosecond(), l.Nanosecond())
			//assert.Equal(t, test.syslog.loc, l.loc)
		})
	}
}
