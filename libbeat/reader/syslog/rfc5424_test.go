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
	"testing"

	"github.com/stretchr/testify/assert"
)

var parseRFC5424Cases = map[string]struct {
	In      string
	Want    message
	WantErr bool
}{
	"example-1": {
		In: "<13>1 2003-08-24T05:14:15.000003-07:00 test-host su 1234 msg-5678 - This is a test message",
		Want: message{
			timestamp: mustParseTime("2003-08-24T05:14:15.000003-07:00"),
			priority:  13,
			facility:  1,
			severity:  5,
			hostname:  "test-host",
			process:   "su",
			pid:       "1234",
			msg:       "This is a test message",
			msgID:     "msg-5678",
			version:   1,
		},
	},
	"example-2": {
		In: `<13>1 2003-08-24T05:14:15.000003-07:00 test-host su 1234 msg-5678 [sd-id-1 foo="bar"] This is a test message`,
		Want: message{
			timestamp: mustParseTime("2003-08-24T05:14:15.000003-07:00"),
			priority:  13,
			facility:  1,
			severity:  5,
			hostname:  "test-host",
			process:   "su",
			pid:       "1234",
			msg:       "This is a test message",
			msgID:     "msg-5678",
			version:   1,
			structuredData: map[string]map[string]string{
				"sd-id-1": {
					"foo": "bar",
				},
			},
		},
	},
	"example-3": {
		In: `<13>1 - - - - - -`,
		Want: message{
			priority: 13,
			facility: 1,
			severity: 5,
			version:  1,
		},
	},
	"example-4": {
		In: `<34>1 2003-10-11T22:14:15.003Z mymachine.example.com su - ID47 - ` + utf8BOM + `'su root' failed for user1 on /dev/pts/8`,
		Want: message{
			timestamp: mustParseTime("2003-10-11T22:14:15.003Z"),
			priority:  34,
			facility:  4,
			severity:  2,
			version:   1,
			hostname:  "mymachine.example.com",
			process:   "su",
			msgID:     "ID47",
			msg:       `'su root' failed for user1 on /dev/pts/8`,
		},
	},
	"example-5": {
		In: `<165>1 2003-10-11T22:14:15.003Z mymachine.example.com evntslog - ID47 [exampleSDID@32473 iut="3" eventSource="Application" eventID="1011"][examplePriority@32473 class="high"]`,
		Want: message{
			timestamp: mustParseTime("2003-10-11T22:14:15.003Z"),
			priority:  165,
			facility:  20,
			severity:  5,
			version:   1,
			hostname:  "mymachine.example.com",
			process:   "evntslog",
			msgID:     "ID47",
			structuredData: map[string]map[string]string{
				"exampleSDID@32473": {
					"iut":         "3",
					"eventSource": "Application",
					"eventID":     "1011",
				},
				"examplePriority@32473": {
					"class": "high",
				},
			},
		},
	},
}

func TestParseRFC5424(t *testing.T) {
	for name, tc := range parseRFC5424Cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, gotErr := parseRFC5424(tc.In)

			if tc.WantErr {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
				assert.Equal(t, tc.Want, got)
			}

			assert.Equal(t, tc.Want, got)
		})
	}
}

func BenchmarkParseRFC5424(b *testing.B) {
	for name, bc := range parseRFC5424Cases {
		bc := bc
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = parseRFC5424(bc.In)
			}
		})
	}
}

var isRFC5424Cases = []struct {
	In   string
	Want bool
	Desc string
}{
	{
		In:   "<13>1 2003-08-24T05:14:15.000003-07:00 test-host su 1234 msg-5678 - This is a test message",
		Want: true,
		Desc: "rfc-5424",
	},
	{
		In:   "<13>Oct 11 22:14:15 test-host this is the message",
		Want: false,
		Desc: "rfc-3164",
	},
	{
		In:   "not a valid message",
		Want: false,
		Desc: "invalid-message",
	},
}

func TestIsRFC5424(t *testing.T) {
	for _, tc := range isRFC5424Cases {
		tc := tc
		t.Run(tc.Desc, func(t *testing.T) {
			t.Parallel()

			got := isRFC5424(tc.In)

			assert.Equal(t, tc.Want, got, tc.Desc)
		})
	}
}

func BenchmarkIsRFC5424(b *testing.B) {
	for _, bc := range isRFC5424Cases {
		bc := bc
		b.Run(bc.Desc, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = isRFC5424(bc.In)
			}
		})
	}
}
