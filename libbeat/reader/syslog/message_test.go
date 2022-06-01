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
		In      string
		InLoc   *time.Location
		Want    time.Time
		WantErr string
	}{
		"bsd-timestamp": {
			In:    "Oct 1 22:04:15",
			InLoc: time.Local,
			Want:  mustParseTime(time.Stamp, "Oct 1 22:04:15", time.Local),
		},
		"loc-nil": {
			In:    "Oct 1 22:04:15",
			InLoc: nil,
			Want:  mustParseTime(time.Stamp, "Oct 1 22:04:15", time.Local),
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
			gotErr := m.setTimestampBSD(tc.In, tc.InLoc)

			if tc.WantErr != "" {
				assert.ErrorContains(t, gotErr, tc.WantErr)
				assert.True(t, m.timestamp.IsZero())
			} else {
				assert.Equal(t, tc.Want, m.timestamp)
			}
		})
	}
}

func TestMessage_SetTimestampRFC3339(t *testing.T) {
	cases := map[string]struct {
		In      string
		Want    time.Time
		WantErr string
	}{
		"rfc3339-timestamp": {
			In:   "1985-04-12T23:20:50.52Z",
			Want: mustParseTime(time.RFC3339Nano, "1985-04-12T23:20:50.52Z", nil),
		},
		"rfc3339-timestamp-with-tz": {
			In:   "1985-04-12T19:20:50.52-04:00",
			Want: mustParseTime(time.RFC3339Nano, "1985-04-12T19:20:50.52-04:00", nil),
		},
		"rfc3339-timestamp-with-milliseconds": {
			In:   "2003-10-11T22:14:15.123Z",
			Want: mustParseTime(time.RFC3339Nano, "2003-10-11T22:14:15.123Z", nil),
		},
		"rfc3339-timestamp-with-microseconds": {
			In:   "2003-10-11T22:14:15.123456Z",
			Want: mustParseTime(time.RFC3339Nano, "2003-10-11T22:14:15.123456Z", nil),
		},
		"rfc3339-timestamp-with-microseconds-with-tz": {
			In:   "2003-10-11T22:14:15.123456-06:00",
			Want: mustParseTime(time.RFC3339Nano, "2003-10-11T22:14:15.123456-06:00", nil),
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
			gotErr := m.setTimestampRFC3339(tc.In)

			if tc.WantErr != "" {
				assert.ErrorContains(t, gotErr, tc.WantErr)
				assert.True(t, m.timestamp.IsZero())
			} else {
				assert.Equal(t, tc.Want, m.timestamp)
			}
		})
	}
}

func TestMessage_SetPriority(t *testing.T) {
	cases := map[string]struct {
		In           string
		WantPriority int
		WantFacility int
		WantSeverity int
		WantErr      string
	}{
		"13": {
			In:           "13",
			WantPriority: 13,
			WantFacility: 1,
			WantSeverity: 5,
		},
		"192": {
			In:      "192",
			WantErr: ErrPriority.Error(),
		},
		"-1": {
			In:      "-1",
			WantErr: ErrPriority.Error(),
		},
		"empty": {
			In:      "",
			WantErr: `invalid priority: strconv.Atoi: parsing "": invalid syntax`,
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			m := message{priority: -1}

			gotErr := m.setPriority(tc.In)

			if tc.WantErr != "" {
				assert.ErrorContains(t, gotErr, tc.WantErr)
				assert.Equal(t, m.priority, -1)
				assert.Zero(t, m.facility)
				assert.Zero(t, m.severity)
			} else {
				assert.NoError(t, gotErr)
				assert.Equal(t, tc.WantPriority, m.priority)
				assert.Equal(t, tc.WantFacility, m.facility)
				assert.Equal(t, tc.WantSeverity, m.severity)
			}
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
		In      string
		Want    int
		WantErr string
	}{
		"valid": {
			In:   "100",
			Want: 100,
		},
		"empty": {
			In:      "",
			WantErr: "invalid version, expected an integer: strconv.Atoi: parsing \"\"",
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var m message

			gotErr := m.setVersion(tc.In)

			if tc.WantErr != "" {
				assert.ErrorContains(t, gotErr, tc.WantErr)
				assert.Zero(t, m.version)
			} else {
				assert.NoError(t, gotErr)
				assert.Equal(t, tc.Want, m.version)
			}

			assert.Equal(t, tc.Want, m.version)
		})
	}
}

func TestMessage_SetRawSDValue(t *testing.T) {
	cases := map[string]struct {
		In   string
		Want string
	}{
		"valid": {
			In:   `[value@1 foo="bar"]`,
			Want: `[value@1 foo="bar"]`,
		},
		"nil-value": {
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

			m.setRawSDValue(tc.In)

			assert.Equal(t, tc.Want, m.rawSDValue)
		})

	}
}

func TestParseStructuredData(t *testing.T) {
	tests := map[string]struct {
		In   string
		Want map[string]interface{}
	}{
		"basic": {
			In: `[value@1 foo="bar"]`,
			Want: map[string]interface{}{
				"value@1": map[string]interface{}{
					"foo": "bar",
				},
			},
		},
		"multi-key": {
			In: `[exampleSDID@32473 iut="3" eventSource="Application" eventID="1011"][examplePriority@32473 class="high"]`,
			Want: map[string]interface{}{
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
			In: `[exampleSDID@32473 iut="3"][exampleSDID@32473 class="high"]`,
			Want: map[string]interface{}{
				"exampleSDID@32473": map[string]interface{}{
					"iut":   "3",
					"class": "high",
				},
			},
		},
		"repeated-id-value": {
			In: `[exampleSDID@32473 class="low"][exampleSDID@32473 class="high"]`,
			Want: map[string]interface{}{
				"exampleSDID@32473": map[string]interface{}{
					"class": "high",
				},
			},
		},
		"non-compliant": {
			In:   `[action:"Drop"; flags:"278528"; ifdir:"inbound"; ifname:"bond1.3999"; loguid:"{0x60928f1d,0x8,0x40de101f,0xfcdbb197}"; origin:"127.0.0.1"; originsicname:"CN=CP,O=cp.com.9jjkfo"; sequencenum:"62"; time:"1620217629"; version:"5"; __policy_id_tag:"product=VPN-1 & FireWall-1[db_tag={F6212FB3-54CE-6344-9164-B224119E2B92};mgmt=cp-m;date=1620031791;policy_name=CP-Cluster]"; action_reason:"Dropped by multiportal infrastructure"; dst:"81.2.69.144"; product:"VPN & FireWall"; proto:"6"; s_port:"52780"; service:"80"; src:"81.2.69.144"]`,
			Want: nil,
		},
		"empty-string": {
			In:   ``,
			Want: nil,
		},
		"nil-value": {
			In:   `-`,
			Want: nil,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := parseStructuredData(tc.In)

			assert.Equal(t, tc.Want, got)
		})
	}
}

func TestMessage_Fields(t *testing.T) {
	cases := map[string]struct {
		In   *message
		Want mapstr.M
	}{
		"valid": {
			In: &message{
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
			Want: mapstr.M{
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
			In: &message{
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
			Want: mapstr.M{
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
			In: &message{
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
			Want: mapstr.M{
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

			got := tc.In.fields()

			assert.Equal(t, tc.Want, got)
		})
	}
}
