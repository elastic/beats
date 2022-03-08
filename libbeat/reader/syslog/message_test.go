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
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
)

func mustParseTime(layout string, value string) time.Time {
	t, err := time.Parse(layout, value)
	if err != nil {
		panic(err)
	}

	return t
}

func mustParseTimeLoc(layout string, value string, loc *time.Location) time.Time {
	t, err := time.ParseInLocation(layout, value, loc)
	if err != nil {
		panic(err)
	}
	if layout == time.Stamp {
		t = t.AddDate(time.Now().Year(), 0, 0)
	}

	return t
}

func TestMessage_SetTimestampBSD(t *testing.T) {
	cases := map[string]struct {
		In    string
		InLoc *time.Location
		Want  time.Time
	}{
		"bsd-timestamp": {
			In:    "Oct 1 22:04:15",
			InLoc: time.Local,
			Want:  mustParseTimeLoc(time.Stamp, "Oct 1 22:04:15", time.Local),
		},
		"loc-nil": {
			In:    "Oct 1 22:04:15",
			InLoc: nil,
			Want:  mustParseTimeLoc(time.Stamp, "Oct 1 22:04:15", time.Local),
		},
		"invalid-timestamp-1": {
			In:    "1985-04-12T23:20:50.52Z",
			InLoc: time.Local,
		},
		"invalid-timestamp-2": {
			In:    "test-value",
			InLoc: time.Local,
		},
		"empty": {
			In: "",
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var m message
			m.setTimestampBSD(tc.In, tc.InLoc)

			assert.Equal(t, tc.Want, m.timestamp)
		})
	}
}

func TestMessage_SetTimestampRFC3339(t *testing.T) {
	cases := map[string]struct {
		In   string
		Want time.Time
	}{
		"rfc3339-timestamp": {
			In:   "1985-04-12T23:20:50.52Z",
			Want: mustParseTime(time.RFC3339Nano, "1985-04-12T23:20:50.52Z"),
		},
		"rfc3339-timestamp-with-tz": {
			In:   "1985-04-12T19:20:50.52-04:00",
			Want: mustParseTime(time.RFC3339Nano, "1985-04-12T19:20:50.52-04:00"),
		},
		"rfc3339-timestamp-with-milliseconds": {
			In:   "2003-10-11T22:14:15.123Z",
			Want: mustParseTime(time.RFC3339Nano, "2003-10-11T22:14:15.123Z"),
		},
		"rfc3339-timestamp-with-microseconds": {
			In:   "2003-10-11T22:14:15.123456Z",
			Want: mustParseTime(time.RFC3339Nano, "2003-10-11T22:14:15.123456Z"),
		},
		"rfc3339-timestamp-with-microseconds-with-tz": {
			In:   "2003-10-11T22:14:15.123456-06:00",
			Want: mustParseTime(time.RFC3339Nano, "2003-10-11T22:14:15.123456-06:00"),
		},
		"invalid-timestamp-1": {
			In: "Oct 1 22:04:15",
		},
		"invalid-timestamp-2": {
			In: "test-value",
		},
		"empty": {
			In: "",
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var m message
			m.setTimestampRFC3339(tc.In)

			assert.Equal(t, tc.Want, m.timestamp)
		})
	}
}

func TestMessage_SetPriority(t *testing.T) {
	cases := map[string]struct {
		In           string
		WantPriority int
		WantFacility int
		WantSeverity int
	}{
		"13": {
			In:           "13",
			WantPriority: 13,
			WantFacility: 1,
			WantSeverity: 5,
		},
		"empty": {
			In: "",
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var m message

			m.setPriority(tc.In)

			assert.Equal(t, tc.WantPriority, m.priority)
			assert.Equal(t, tc.WantFacility, m.facility)
			assert.Equal(t, tc.WantSeverity, m.severity)
		})
	}
}

func TestMessage_SetHostname(t *testing.T) {
	cases := map[string]struct {
		In   string
		Want string
	}{
		"valid": {
			In:   "test-value",
			Want: "test-value",
		},
		"dash-ignored": {
			In: "-",
		},
		"empty": {
			In: "",
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			var m message

			m.setHostname(tc.In)

			assert.Equal(t, tc.Want, m.hostname)
		})
	}
}

func TestMessage_SetMsg(t *testing.T) {
	cases := map[string]struct {
		In   string
		Want string
	}{
		"valid": {
			In:   "test-value",
			Want: "test-value",
		},
		"empty": {
			In: "",
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var m message

			m.setMsg(tc.In)

			assert.Equal(t, tc.Want, m.msg)
		})
	}
}

func TestMessage_SetTag(t *testing.T) {
	cases := map[string]struct {
		In   string
		Want string
		Name string
	}{
		"valid": {
			In:   "test-value",
			Want: "test-value",
		},
		"empty": {
			In: "",
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			var m message

			m.setTag(tc.In)

			assert.Equal(t, tc.Want, m.process)
		})
	}
}

func TestMessage_SetAppName(t *testing.T) {
	cases := map[string]struct {
		In   string
		Want string
	}{
		"valid": {
			In:   "test-value",
			Want: "test-value",
		},
		"dash-ignored": {
			In: "-",
		},
		"empty": {
			In: "",
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var m message

			m.setAppName(tc.In)

			assert.Equal(t, tc.Want, m.process)
		})

	}
}

func TestMessage_SetContent(t *testing.T) {
	cases := map[string]struct {
		In   string
		Want string
		Name string
	}{
		"valid": {
			In:   "test-value",
			Want: "test-value",
		},
		"empty": {
			In: "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			var m message

			m.setContent(tc.In)

			assert.Equal(t, tc.Want, m.pid)
		})
	}
}

func TestMessage_SetProcID(t *testing.T) {
	cases := map[string]struct {
		In   string
		Want string
	}{
		"valid": {
			In:   "test-value",
			Want: "test-value",
		},
		"dash-ignored": {
			In: "-",
		},
		"empty": {
			In: "",
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var m message

			m.setProcID(tc.In)

			assert.Equal(t, tc.Want, m.pid)
		})
	}
}

func TestMessage_SetMsgID(t *testing.T) {
	cases := map[string]struct {
		In   string
		Want string
	}{
		"valid": {
			In:   "test-value",
			Want: "test-value",
		},
		"empty": {
			In: "",
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var m message

			m.setMsgID(tc.In)

			assert.Equal(t, tc.Want, m.msgID)
		})

	}
}

func TestMessage_SetVersion(t *testing.T) {
	cases := map[string]struct {
		In   string
		Want int
	}{
		"valid": {
			In:   "100",
			Want: 100,
		},
		"empty": {
			In: "",
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var m message

			m.setVersion(tc.In)

			assert.Equal(t, tc.Want, m.version)
		})
	}
}

func TestMessage_SetData(t *testing.T) {
	tests := map[string]struct {
		Data    map[string]map[string]string
		InID    string
		InKey   string
		InValue string
		Want    map[string]map[string]string
	}{
		"ok": {
			Data: map[string]map[string]string{
				"A": {},
			},
			InID:    "A",
			InKey:   "B",
			InValue: "foobar",
			Want: map[string]map[string]string{
				"A": {
					"B": "foobar",
				},
			},
		},
		"overwrite": {
			Data: map[string]map[string]string{
				"A": {
					"B": "C",
				},
			},
			InID:    "A",
			InKey:   "B",
			InValue: "foobar",
			Want: map[string]map[string]string{
				"A": {
					"B": "foobar",
				},
			},
		},
		"missing-id": {
			Data:    map[string]map[string]string{},
			InID:    "A",
			InKey:   "B",
			InValue: "foobar",
			Want:    map[string]map[string]string{},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			m := message{
				structuredData: tc.Data,
			}

			m.setDataValue(tc.InID, tc.InKey, tc.InValue)

			assert.Equal(t, tc.Want, m.structuredData)
		})
	}
}

func TestMessage_Fields(t *testing.T) {
	cases := map[string]struct {
		In   *message
		Want common.MapStr
	}{
		"valid": {
			In: &message{
				timestamp: mustParseTime(time.RFC3339Nano, "2003-10-11T22:14:15.123456-06:00"),
				facility:  1,
				severity:  5,
				priority:  13,
				hostname:  "test-host",
				msg:       "this is a test message",
				process:   "su",
				pid:       "1024",
				msgID:     "msg123",
				version:   1,
				structuredData: map[string]map[string]string{
					"a": {
						"b": "c",
					},
				},
			},
			Want: common.MapStr{
				"log": common.MapStr{
					"syslog": common.MapStr{
						"priority": 13,
						"facility": common.MapStr{
							"code": 1,
							"name": "user-level",
						},
						"severity": common.MapStr{
							"code": 5,
							"name": "Notice",
						},
						"hostname": "test-host",
						"appname":  "su",
						"procid":   "1024",
						"msgid":    "msg123",
						"version":  1,
						"structured_data": map[string]map[string]string{
							"a": {
								"b": "c",
							},
						},
					},
				},
				"message": "this is a test message",
			},
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := tc.In.fields()

			assert.Equal(t, tc.Want, got)
		})
	}
}
