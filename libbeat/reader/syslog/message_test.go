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

func TestMessage_SetTimestampBSD(t *testing.T) {
	cases := map[string]struct {
		in      string
		inLoc   *time.Location
		want    time.Time
		wantErr string
	}{
		"bsd-timestamp": {
			in:    "Oct 1 22:04:15",
			inLoc: time.Local,
			want:  mustParseTime(time.Stamp, "Oct 1 22:04:15", time.Local),
		},
		"loc-nil": {
			in:    "Oct 1 22:04:15",
			inLoc: nil,
			want:  mustParseTime(time.Stamp, "Oct 1 22:04:15", time.Local),
		},
		"invalid-timestamp-1": {
			in:      "1985-04-12T23:20:50.52Z",
			inLoc:   time.Local,
			wantErr: `parsing time "1985-04-12T23:20:50.52Z" as "Jan _2 15:04:05": cannot parse "1985-04-12T23:20:50.52Z" as "Jan"`,
		},
		"invalid-timestamp-2": {
			in:      "test-value",
			inLoc:   time.Local,
			wantErr: `parsing time "test-value" as "Jan _2 15:04:05": cannot parse "test-value" as "Jan"`,
		},
		"empty": {
			in:      "",
			wantErr: `parsing time "" as "Jan _2 15:04:05": cannot parse "" as "Jan"`,
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var m message
			gotErr := m.setTimestampBSD(tc.in, tc.inLoc)

			if tc.wantErr != "" {
				assert.ErrorContains(t, gotErr, tc.wantErr)
				assert.True(t, m.timestamp.IsZero())
			} else {
				assert.Equal(t, tc.want, m.timestamp)
				assert.NoError(t, gotErr)
			}
		})
	}
}

func TestMessage_SetTimestampRFC3339(t *testing.T) {
	cases := map[string]struct {
		in      string
		want    time.Time
		wantErr string
	}{
		"rfc3339-timestamp": {
			in:   "1985-04-12T23:20:50.52Z",
			want: mustParseTime(time.RFC3339Nano, "1985-04-12T23:20:50.52Z", nil),
		},
		"rfc3339-timestamp-with-tz": {
			in:   "1985-04-12T19:20:50.52-04:00",
			want: mustParseTime(time.RFC3339Nano, "1985-04-12T19:20:50.52-04:00", nil),
		},
		"rfc3339-timestamp-with-milliseconds": {
			in:   "2003-10-11T22:14:15.123Z",
			want: mustParseTime(time.RFC3339Nano, "2003-10-11T22:14:15.123Z", nil),
		},
		"rfc3339-timestamp-with-microseconds": {
			in:   "2003-10-11T22:14:15.123456Z",
			want: mustParseTime(time.RFC3339Nano, "2003-10-11T22:14:15.123456Z", nil),
		},
		"rfc3339-timestamp-with-microseconds-with-tz": {
			in:   "2003-10-11T22:14:15.123456-06:00",
			want: mustParseTime(time.RFC3339Nano, "2003-10-11T22:14:15.123456-06:00", nil),
		},
		"invalid-timestamp-1": {
			in:      "Oct 1 22:04:15",
			wantErr: `parsing time "Oct 1 22:04:15" as "2006-01-02T15:04:05.999999999Z07:00": cannot parse "Oct 1 22:04:15" as "2006"`,
		},
		"invalid-timestamp-2": {
			in:      "test-value",
			wantErr: `parsing time "test-value" as "2006-01-02T15:04:05.999999999Z07:00": cannot parse "test-value" as "2006"`,
		},
		"empty": {
			in:      "",
			wantErr: `parsing time "" as "2006-01-02T15:04:05.999999999Z07:00": cannot parse "" as "2006"`,
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var m message
			gotErr := m.setTimestampRFC3339(tc.in)

			if tc.wantErr != "" {
				assert.ErrorContains(t, gotErr, tc.wantErr)
				assert.True(t, m.timestamp.IsZero())
			} else {
				assert.Equal(t, tc.want, m.timestamp)
				assert.NoError(t, gotErr)
			}
		})
	}
}

func TestMessage_SetPriority(t *testing.T) {
	cases := map[string]struct {
		in           string
		wantPriority int
		wantFacility int
		wantSeverity int
		wantErr      string
	}{
		"13": {
			in:           "13",
			wantPriority: 13,
			wantFacility: 1,
			wantSeverity: 5,
		},
		"192": {
			in:      "192",
			wantErr: ErrPriority.Error(),
		},
		"-1": {
			in:      "-1",
			wantErr: ErrPriority.Error(),
		},
		"empty": {
			in:      "",
			wantErr: `invalid priority: strconv.Atoi: parsing "": invalid syntax`,
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			m := message{priority: -1}

			gotErr := m.setPriority(tc.in)

			if tc.wantErr != "" {
				assert.ErrorContains(t, gotErr, tc.wantErr)
				assert.Equal(t, m.priority, -1)
				assert.Zero(t, m.facility)
				assert.Zero(t, m.severity)
			} else {
				assert.NoError(t, gotErr)
				assert.Equal(t, tc.wantPriority, m.priority)
				assert.Equal(t, tc.wantFacility, m.facility)
				assert.Equal(t, tc.wantSeverity, m.severity)
			}
		})
	}
}

func TestMessage_SetHostname(t *testing.T) {
	cases := map[string]struct {
		in   string
		want string
	}{
		"valid": {
			in:   "test-value",
			want: "test-value",
		},
		"dash-ignored": {
			in:   "-",
			want: "",
		},
		"empty": {
			in:   "",
			want: "",
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			var m message

			m.setHostname(tc.in)

			assert.Equal(t, tc.want, m.hostname)
		})
	}
}

func TestMessage_SetMsg(t *testing.T) {
	cases := map[string]struct {
		in   string
		want string
	}{
		"valid": {
			in:   "test-value",
			want: "test-value",
		},
		"empty": {
			in:   "",
			want: "",
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var m message

			m.setMsg(tc.in)

			assert.Equal(t, tc.want, m.msg)
		})
	}
}

func TestMessage_SetTag(t *testing.T) {
	cases := map[string]struct {
		in   string
		want string
	}{
		"valid": {
			in:   "test-value",
			want: "test-value",
		},
		"empty": {
			in:   "",
			want: "",
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			var m message

			m.setTag(tc.in)

			assert.Equal(t, tc.want, m.process)
		})
	}
}

func TestMessage_SetAppName(t *testing.T) {
	cases := map[string]struct {
		in   string
		want string
	}{
		"valid": {
			in:   "test-value",
			want: "test-value",
		},
		"dash-ignored": {
			in:   "-",
			want: "",
		},
		"empty": {
			in:   "",
			want: "",
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var m message

			m.setAppName(tc.in)

			assert.Equal(t, tc.want, m.process)
		})

	}
}

func TestMessage_SetContent(t *testing.T) {
	cases := map[string]struct {
		in   string
		want string
	}{
		"valid": {
			in:   "test-value",
			want: "test-value",
		},
		"empty": {
			in:   "",
			want: "",
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var m message

			m.setContent(tc.in)

			assert.Equal(t, tc.want, m.pid)
		})
	}
}

func TestMessage_SetProcID(t *testing.T) {
	cases := map[string]struct {
		in   string
		want string
	}{
		"valid": {
			in:   "test-value",
			want: "test-value",
		},
		"dash-ignored": {
			in:   "-",
			want: "",
		},
		"empty": {
			in:   "",
			want: "",
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var m message

			m.setProcID(tc.in)

			assert.Equal(t, tc.want, m.pid)
		})
	}
}

func TestMessage_SetMsgID(t *testing.T) {
	cases := map[string]struct {
		in   string
		want string
	}{
		"valid": {
			in:   "test-value",
			want: "test-value",
		},
		"empty": {
			in:   "",
			want: "",
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var m message

			m.setMsgID(tc.in)

			assert.Equal(t, tc.want, m.msgID)
		})

	}
}

func TestMessage_SetVersion(t *testing.T) {
	cases := map[string]struct {
		in      string
		want    int
		wantErr string
	}{
		"valid": {
			in:   "100",
			want: 100,
		},
		"empty": {
			in:      "",
			wantErr: "invalid version, expected an integer: strconv.Atoi: parsing \"\"",
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var m message

			gotErr := m.setVersion(tc.in)

			if tc.wantErr != "" {
				assert.ErrorContains(t, gotErr, tc.wantErr)
				assert.Zero(t, m.version)
			} else {
				assert.NoError(t, gotErr)
				assert.Equal(t, tc.want, m.version)
			}

			assert.Equal(t, tc.want, m.version)
		})
	}
}

func TestMessage_SetRawSDValue(t *testing.T) {
	cases := map[string]struct {
		in   string
		want string
	}{
		"valid": {
			in:   `[value@1 foo="bar"]`,
			want: `[value@1 foo="bar"]`,
		},
		"nil-value": {
			in:   "-",
			want: "",
		},
		"empty": {
			in:   "",
			want: "",
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var m message

			m.setRawSDValue(tc.in)

			assert.Equal(t, tc.want, m.rawSDValue)
		})

	}
}

func TestParseStructuredData(t *testing.T) {
	tests := map[string]struct {
		in   string
		want map[string]interface{}
	}{
		"basic": {
			in: `[value@1 foo="bar"]`,
			want: map[string]interface{}{
				"value@1": map[string]interface{}{
					"foo": "bar",
				},
			},
		},
		"multi-key": {
			in: `[exampleSDID@32473 iut="3" eventSource="Application" eventID="1011"][examplePriority@32473 class="high"]`,
			want: map[string]interface{}{
				"exampleSDID@32473": map[string]interface{}{
					"iut":         "3",
					"eventSource": "Application",
					"eventID":     "1011",
				},
				"examplePriority@32473": map[string]interface{}{
					"class": "high",
				},
			},
		},
		"repeated-id": {
			in: `[exampleSDID@32473 iut="3"][exampleSDID@32473 class="high"]`,
			want: map[string]interface{}{
				"exampleSDID@32473": map[string]interface{}{
					"iut":   "3",
					"class": "high",
				},
			},
		},
		"repeated-id-value": {
			in: `[exampleSDID@32473 class="low"][exampleSDID@32473 class="high"]`,
			want: map[string]interface{}{
				"exampleSDID@32473": map[string]interface{}{
					"class": "high",
				},
			},
		},
		"non-compliant": {
			in:   `[action:"Drop"; flags:"278528"; ifdir:"inbound"; ifname:"bond1.3999"; loguid:"{0x60928f1d,0x8,0x40de101f,0xfcdbb197}"; origin:"127.0.0.1"; originsicname:"CN=CP,O=cp.com.9jjkfo"; sequencenum:"62"; time:"1620217629"; version:"5"; __policy_id_tag:"product=VPN-1 & FireWall-1[db_tag={F6212FB3-54CE-6344-9164-B224119E2B92};mgmt=cp-m;date=1620031791;policy_name=CP-Cluster]"; action_reason:"Dropped by multiportal infrastructure"; dst:"81.2.69.144"; product:"VPN & FireWall"; proto:"6"; s_port:"52780"; service:"80"; src:"81.2.69.144"]`,
			want: nil,
		},
		"empty-string": {
			in:   ``,
			want: nil,
		},
		"nil-value": {
			in:   `-`,
			want: nil,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := parseStructuredData(tc.in)

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestMessage_Fields(t *testing.T) {
	cases := map[string]struct {
		in   *message
		want mapstr.M
	}{
		"valid": {
			in: &message{
				timestamp:  mustParseTime(time.RFC3339Nano, "2003-10-11T22:14:15.123456-06:00", nil),
				facility:   1,
				severity:   5,
				priority:   13,
				hostname:   "test-host",
				msg:        "this is a test message",
				process:    "su",
				pid:        "1024",
				msgID:      "msg123",
				version:    1,
				rawSDValue: `[a b="c"]`,
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
						"msgid":    "msg123",
						"version":  "1",
						"structured_data": map[string]interface{}{
							"a": map[string]interface{}{
								"b": "c",
							},
						},
					},
				},
				"message": "this is a test message",
			},
		},
		"non-compliant-sd": {
			in: &message{
				timestamp:  mustParseTime(time.RFC3339Nano, "2003-10-11T22:14:15.123456-06:00", nil),
				facility:   1,
				severity:   5,
				priority:   13,
				hostname:   "test-host",
				process:    "su",
				pid:        "1024",
				msgID:      "msg123",
				version:    1,
				rawSDValue: `[action:"Drop"; flags:"278528"; ifdir:"inbound"; ifname:"bond1.3999"; loguid:"{0x60928f1d,0x8,0x40de101f,0xfcdbb197}"; origin:"127.0.0.1"; originsicname:"CN=CP,O=cp.com.9jjkfo"; sequencenum:"62"; time:"1620217629"; version:"5"; __policy_id_tag:"product=VPN-1 & FireWall-1[db_tag={F6212FB3-54CE-6344-9164-B224119E2B92};mgmt=cp-m;date=1620031791;policy_name=CP-Cluster]"; action_reason:"Dropped by multiportal infrastructure"; dst:"81.2.69.144"; product:"VPN & FireWall"; proto:"6"; s_port:"52780"; service:"80"; src:"81.2.69.144"]`,
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
						"msgid":    "msg123",
						"version":  "1",
					},
				},
				"message": `[action:"Drop"; flags:"278528"; ifdir:"inbound"; ifname:"bond1.3999"; loguid:"{0x60928f1d,0x8,0x40de101f,0xfcdbb197}"; origin:"127.0.0.1"; originsicname:"CN=CP,O=cp.com.9jjkfo"; sequencenum:"62"; time:"1620217629"; version:"5"; __policy_id_tag:"product=VPN-1 & FireWall-1[db_tag={F6212FB3-54CE-6344-9164-B224119E2B92};mgmt=cp-m;date=1620031791;policy_name=CP-Cluster]"; action_reason:"Dropped by multiportal infrastructure"; dst:"81.2.69.144"; product:"VPN & FireWall"; proto:"6"; s_port:"52780"; service:"80"; src:"81.2.69.144"]`,
			},
		},
		"non-compliant-sd-with-msg": {
			in: &message{
				timestamp:  mustParseTime(time.RFC3339Nano, "2003-10-11T22:14:15.123456-06:00", nil),
				facility:   1,
				severity:   5,
				priority:   13,
				hostname:   "test-host",
				process:    "su",
				pid:        "1024",
				msgID:      "msg123",
				version:    1,
				msg:        "This is a test message",
				rawSDValue: `[action:"Drop"; flags:"278528"; ifdir:"inbound"; ifname:"bond1.3999"; loguid:"{0x60928f1d,0x8,0x40de101f,0xfcdbb197}"; origin:"127.0.0.1"; originsicname:"CN=CP,O=cp.com.9jjkfo"; sequencenum:"62"; time:"1620217629"; version:"5"; __policy_id_tag:"product=VPN-1 & FireWall-1[db_tag={F6212FB3-54CE-6344-9164-B224119E2B92};mgmt=cp-m;date=1620031791;policy_name=CP-Cluster]"; action_reason:"Dropped by multiportal infrastructure"; dst:"81.2.69.144"; product:"VPN & FireWall"; proto:"6"; s_port:"52780"; service:"80"; src:"81.2.69.144"]`,
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
						"msgid":    "msg123",
						"version":  "1",
					},
				},
				"message": `[action:"Drop"; flags:"278528"; ifdir:"inbound"; ifname:"bond1.3999"; loguid:"{0x60928f1d,0x8,0x40de101f,0xfcdbb197}"; origin:"127.0.0.1"; originsicname:"CN=CP,O=cp.com.9jjkfo"; sequencenum:"62"; time:"1620217629"; version:"5"; __policy_id_tag:"product=VPN-1 & FireWall-1[db_tag={F6212FB3-54CE-6344-9164-B224119E2B92};mgmt=cp-m;date=1620031791;policy_name=CP-Cluster]"; action_reason:"Dropped by multiportal infrastructure"; dst:"81.2.69.144"; product:"VPN & FireWall"; proto:"6"; s_port:"52780"; service:"80"; src:"81.2.69.144"] This is a test message`,
			},
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := tc.in.fields()

			assert.Equal(t, tc.want, got)
		})
	}
}
