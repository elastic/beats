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

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgtype"
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
		t = t.AddDate(time.Now().In(loc).Year(), 0, 0)
	}

	return t
}

var syslogCases = map[string]struct {
	Cfg      *common.Config
	In       common.MapStr
	Want     common.MapStr
	WantTime time.Time
	WantErr  bool
}{
	"rfc-3164": {
		Cfg: common.MustNewConfigFrom(common.MapStr{
			"timezone": "America/Chicago",
		}),
		In: common.MapStr{
			"message": `<13>Oct 11 22:14:15 test-host su[1024]: this is the message`,
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
				},
			},
			"event": common.MapStr{
				"severity": 5,
				"original": `<13>Oct 11 22:14:15 test-host su[1024]: this is the message`,
			},
			"message": "this is the message",
		},
		WantTime: mustParseTimeLoc(time.Stamp, "Oct 11 22:14:15", cfgtype.MustNewTimezone("America/Chicago").Location()),
	},
	"rfc-5424": {
		Cfg: common.MustNewConfigFrom(common.MapStr{}),
		In: common.MapStr{
			"message": `<165>1 2003-10-11T22:14:15.003Z mymachine.example.com evntslog 1024 ID47 [exampleSDID@32473 iut="3" eventSource="Application" eventID="1011"][examplePriority@32473 class="high"] this is the message`,
		},
		Want: common.MapStr{
			"log": common.MapStr{
				"syslog": common.MapStr{
					"priority": 165,
					"facility": common.MapStr{
						"code": 20,
						"name": "local4",
					},
					"severity": common.MapStr{
						"code": 5,
						"name": "Notice",
					},
					"hostname": "mymachine.example.com",
					"appname":  "evntslog",
					"procid":   "1024",
					"msgid":    "ID47",
					"version":  1,
					"data": map[string]map[string]string{
						"examplePriority@32473": {
							"class": "high",
						},
						"exampleSDID@32473": {
							"eventID":     "1011",
							"eventSource": "Application",
							"iut":         "3",
						},
					},
				},
			},
			"event": common.MapStr{
				"severity": 5,
				"original": `<165>1 2003-10-11T22:14:15.003Z mymachine.example.com evntslog 1024 ID47 [exampleSDID@32473 iut="3" eventSource="Application" eventID="1011"][examplePriority@32473 class="high"] this is the message`,
			},
			"message": "this is the message",
		},
		WantTime: mustParseTime(time.RFC3339Nano, "2003-10-11T22:14:15.003Z"),
	},
}

func TestSyslog(t *testing.T) {
	for name, tc := range syslogCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			p, err := New(tc.Cfg)
			if err != nil {
				panic(err)
			}
			event := &beat.Event{
				Fields: tc.In,
			}

			got, gotErr := p.Run(event)
			if tc.WantErr {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
			}

			assert.Equal(t, tc.Want, got.Fields)
		})
	}
}

func BenchmarkSyslog(b *testing.B) {
	for name, bc := range syslogCases {
		bc := bc
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {

				p, _ := New(bc.Cfg)
				event := &beat.Event{
					Fields: bc.In,
				}

				_, _ = p.Run(event)
			}
		})
	}
}

func TestAppendStringField(t *testing.T) {
	tests := map[string]struct {
		InMap   common.MapStr
		InField string
		InValue string
		Want    common.MapStr
	}{
		"nil": {
			InMap:   common.MapStr{},
			InField: "error",
			InValue: "foo",
			Want: common.MapStr{
				"error": "foo",
			},
		},
		"string": {
			InMap: common.MapStr{
				"error": "foo",
			},
			InField: "error",
			InValue: "bar",
			Want: common.MapStr{
				"error": []string{"foo", "bar"},
			},
		},
		"string-slice": {
			InMap: common.MapStr{
				"error": []string{"foo", "bar"},
			},
			InField: "error",
			InValue: "some value",
			Want: common.MapStr{
				"error": []string{"foo", "bar", "some value"},
			},
		},
		"interface-slice": {
			InMap: common.MapStr{
				"error": []interface{}{"foo", "bar"},
			},
			InField: "error",
			InValue: "some value",
			Want: common.MapStr{
				"error": []interface{}{"foo", "bar", "some value"},
			},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			appendStringField(tc.InMap, tc.InField, tc.InValue)

			assert.Equal(t, tc.Want, tc.InMap)
		})
	}
}
