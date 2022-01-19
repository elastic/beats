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
			"syslog": common.MapStr{
				"facility":       1,
				"priority":       13,
				"severity":       5,
				"facility_label": "user-level",
				"severity_label": "Notice",
			},
			"event": common.MapStr{
				"severity": 5,
				"original": `<13>Oct 11 22:14:15 test-host su[1024]: this is the message`,
			},
			"process": common.MapStr{
				"name": "su",
				"pid":  "1024",
			},
			"host": common.MapStr{
				"name": "test-host",
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
			"syslog": common.MapStr{
				"priority":       165,
				"facility":       20,
				"severity":       5,
				"facility_label": "local4",
				"severity_label": "Notice",
				"msgid":          "ID47",
				"version":        1,
				"data": map[string]map[string]string{
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
			"event": common.MapStr{
				"severity": 5,
				"original": `<165>1 2003-10-11T22:14:15.003Z mymachine.example.com evntslog 1024 ID47 [exampleSDID@32473 iut="3" eventSource="Application" eventID="1011"][examplePriority@32473 class="high"] this is the message`,
			},
			"process": common.MapStr{
				"name": "evntslog",
				"pid":  "1024",
			},
			"host": common.MapStr{
				"name": "mymachine.example.com",
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
