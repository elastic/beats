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

const PriorityTestTemplate = `<%d>1 2016-02-21T04:32:57+00:00 web1 someservice - - [origin x-service="someservice"][meta sequenceId="14125553"] 127.0.0.1 - - 1456029177 "GET /v1/ok HTTP/1.1" 200 145 "-" "hacheck 0.9.0" 24306 127.0.0.1:40124 575`;

const VersionTestTemplate = `<190>%d 2016-02-21T04:32:57+00:00 web1 someservice - - [origin x-service="someservice"][meta sequenceId="14125553"] 127.0.0.1 - - 1456029177 "GET /v1/ok HTTP/1.1" 200 145 "-" "hacheck 0.9.0" 24306 127.0.0.1:40124 575`;

type testRule struct {
	title  string
	log    []byte
	syslog event
}

func createVersionTestRule(v int) testRule {
	return testRule{
		title: fmt.Sprintf("versionTest v:%d", v),
		log:   []byte(fmt.Sprintf(VersionTestTemplate, v)),
		syslog: event{
			version:  v,
			priority: 190,
		},
	}
}
func TestParseRfc5424Syslog(t *testing.T) {
	tests := []testRule{
		//{
		//	title: "priority test",
		//	log:   []byte(fmt.Sprintf(PriorityTestTemplate, 0)),
		//	syslog: event{
		//		priority: 0,
		//		version:  1,
		//	},
		//},
		//{
		//	title: "priority test",
		//	log:   []byte(fmt.Sprintf(PriorityTestTemplate, 190)),
		//	syslog: event{
		//		priority: 190,
		//		version:  1,
		//	},
		//},
		//{
		//	title: "priority test",
		//	log:   []byte(`<190>1 2016-02-21T04:32:57+00:00 web1 someservice - - [origin x-service="someservice"][meta sequenceId="14125553"] 127.0.0.1 - - 1456029177 "GET /v1/ok HTTP/1.1" 200 145 "-" "hacheck 0.9.0" 24306 127.0.0.1:40124 575`),
		//	syslog: event{
		//		priority: 190,
		//		version:  1,
		//	},
		//},
		//{
		//	title: "priority test",
		//	log:   []byte(`<191>1 2016-02-21T04:32:57+00:00 web1 someservice - - [origin x-service="someservice"][meta sequenceId="14125553"] 127.0.0.1 - - 1456029177 "GET /v1/ok HTTP/1.1" 200 145 "-" "hacheck 0.9.0" 24306 127.0.0.1:40124 575`),
		//	syslog: event{
		//		priority: 191,
		//		version:  1,
		//	},
		//},
		//{
		//	title: "priority test",
		//	log:   []byte(`<192>1 2016-02-21T04:32:57+00:00 web1 someservice - - [origin x-service="someservice"][meta sequenceId="14125553"] 127.0.0.1 - - 1456029177 "GET /v1/ok HTTP/1.1" 200 145 "-" "hacheck 0.9.0" 24306 127.0.0.1:40124 575`),
		//	syslog: event{
		//		priority: 192,
		//		version:  1,
		//	},
		//},
		{
			title: "priority test",
			log:   []byte(`<42>10 2016-02-21T04:32:57+00:00 web1 someservice - - [origin x-service="someservice"][meta sequenceId="14125553"] 127.0.0.1 - - 1456029177 "GET /v1/ok HTTP/1.1" 200 145 "-" "hacheck 0.9.0" 24306 127.0.0.1:40124 575`),
			syslog: event{
				priority: 42,
				version:  10,
			},
		},
	}

	// add some version test
	//tests = append(tests, createVersionTestRule(0))
	//tests = append(tests, createVersionTestRule(100))
	//tests = append(tests, createVersionTestRule(1000))

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
