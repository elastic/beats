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
	"github.com/elastic/beats/v7/libbeat/common/cfgtype"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// mustParseTime will parse value into a time.Time using the provided layout. If value
// cannot be parsed, this function will panic. If layout does not specify a time zone,
// then a time.Location should be provided by loc. If layout does specify a time zone,
// then loc should be nil. Layouts that do not specify a year will be enriched with
// the current year relative to the location specified for the parsed timestamp.
func mustParseTime(layout, value string, loc *time.Location) time.Time {
	var t time.Time
	var err error

	if loc != nil {
		t, err = time.ParseInLocation(layout, value, loc)
	} else {
		t, err = time.Parse(layout, value)
	}
	if err != nil {
		panic(err)
	}

	// Timestamps that do not include a year will be enriched using the
	// current year relative to the location specified for the timestamp.
	if t.Year() == 0 {
		t = t.AddDate(time.Now().In(t.Location()).Year(), 0, 0)
	}

	return t
}

var syslogCases = map[string]struct {
	cfg      *conf.C
	in       mapstr.M
	want     mapstr.M
	wantTime time.Time
	wantErr  bool
}{
	"rfc-3164": {
		cfg: conf.MustNewConfigFrom(mapstr.M{
			"timezone": "America/Chicago",
		}),
		in: mapstr.M{
			"message": `<13>Oct 11 22:14:15 test-host su[1024]: this is the message`,
		},
		want: mapstr.M{
			"log": mapstr.M{
				"syslog": mapstr.M{
					"priority": 13,
					"facility": mapstr.M{
						"code": 1,
						"name": "user-level",
					},
					"severity": mapstr.M{
						"code": 5,
						"name": "Notice",
					},
					"hostname": "test-host",
					"appname":  "su",
					"procid":   "1024",
				},
			},
			"message": "this is the message",
		},
		wantTime: mustParseTime(time.Stamp, "Oct 11 22:14:15", cfgtype.MustNewTimezone("America/Chicago").Location()),
	},
	"rfc-5424": {
		cfg: conf.MustNewConfigFrom(mapstr.M{}),
		in: mapstr.M{
			"message": `<165>1 2003-10-11T22:14:15.003Z mymachine.example.com evntslog 1024 ID47 [exampleSDID@32473 iut="3" eventSource="Application" eventID="1011"][examplePriority@32473 class="high"] this is the message`,
		},
		want: mapstr.M{
			"log": mapstr.M{
				"syslog": mapstr.M{
					"priority": 165,
					"facility": mapstr.M{
						"code": 20,
						"name": "local4",
					},
					"severity": mapstr.M{
						"code": 5,
						"name": "Notice",
					},
					"hostname": "mymachine.example.com",
					"appname":  "evntslog",
					"procid":   "1024",
					"msgid":    "ID47",
					"version":  "1",
					"structured_data": map[string]interface{}{
						"examplePriority@32473": map[string]interface{}{
							"class": "high",
						},
						"exampleSDID@32473": map[string]interface{}{
							"eventID":     "1011",
							"eventSource": "Application",
							"iut":         "3",
						},
					},
				},
			},
			"message": "this is the message",
		},
		wantTime: mustParseTime(time.RFC3339Nano, "2003-10-11T22:14:15.003Z", nil),
	},
}

func TestSyslog(t *testing.T) {
	for name, tc := range syslogCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			p, err := New(tc.cfg)
			if err != nil {
				panic(err)
			}
			event := &beat.Event{
				Fields: tc.in,
			}

			got, gotErr := p.Run(event)
			if tc.wantErr {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
			}

			assert.Equal(t, tc.want, got.Fields)
		})
	}
}

func BenchmarkSyslog(b *testing.B) {
	for name, bc := range syslogCases {
		bc := bc
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {

				p, _ := New(bc.cfg)
				event := &beat.Event{
					Fields: bc.in,
				}

				_, _ = p.Run(event)
			}
		})
	}
}

func TestAppendStringField(t *testing.T) {
	tests := map[string]struct {
		inMap   mapstr.M
		inField string
		inValue string
		want    mapstr.M
	}{
		"nil": {
			inMap:   mapstr.M{},
			inField: "error",
			inValue: "foo",
			want: mapstr.M{
				"error": "foo",
			},
		},
		"string": {
			inMap: mapstr.M{
				"error": "foo",
			},
			inField: "error",
			inValue: "bar",
			want: mapstr.M{
				"error": []string{"foo", "bar"},
			},
		},
		"string-slice": {
			inMap: mapstr.M{
				"error": []string{"foo", "bar"},
			},
			inField: "error",
			inValue: "some value",
			want: mapstr.M{
				"error": []string{"foo", "bar", "some value"},
			},
		},
		"interface-slice": {
			inMap: mapstr.M{
				"error": []interface{}{"foo", "bar"},
			},
			inField: "error",
			inValue: "some value",
			want: mapstr.M{
				"error": []interface{}{"foo", "bar", "some value"},
			},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			appendStringField(tc.inMap, tc.inField, tc.inValue)

			assert.Equal(t, tc.want, tc.inMap)
		})
	}
}
