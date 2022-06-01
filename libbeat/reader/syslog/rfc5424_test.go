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
)

func TestParseRFC5424(t *testing.T) {
	tests := map[string]struct {
		In      string
		Want    message
		WantErr string
	}{
		"example-1": {
			In: "<13>1 2003-08-24T05:14:15.000003-07:00 test-host su 1234 msg-5678 - This is a test message",
			Want: message{
				timestamp: mustParseTime(time.RFC3339Nano, "2003-08-24T05:14:15.000003-07:00", nil),
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
				timestamp:  mustParseTime(time.RFC3339Nano, "2003-08-24T05:14:15.000003-07:00", nil),
				priority:   13,
				facility:   1,
				severity:   5,
				hostname:   "test-host",
				process:    "su",
				pid:        "1234",
				msg:        "This is a test message",
				msgID:      "msg-5678",
				version:    1,
				rawSDValue: `[sd-id-1 foo="bar"]`,
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
				timestamp: mustParseTime(time.RFC3339Nano, "2003-10-11T22:14:15.003Z", nil),
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
				timestamp:  mustParseTime(time.RFC3339Nano, "2003-10-11T22:14:15.003Z", nil),
				priority:   165,
				facility:   20,
				severity:   5,
				version:    1,
				hostname:   "mymachine.example.com",
				process:    "evntslog",
				msgID:      "ID47",
				rawSDValue: `[exampleSDID@32473 iut="3" eventSource="Application" eventID="1011"][examplePriority@32473 class="high"]`,
			},
		},
		"non-compliant-sd": {
			In: `<165>1 2003-10-11T22:14:15.003Z mymachine.example.com evntslog - ID47 [action:"Drop"; flags:"278528"; ifdir:"inbound"; ifname:"bond1.3999"; loguid:"{0x60928f1d,0x8,0x40de101f,0xfcdbb197}"; origin:"127.0.0.1"; originsicname:"CN=CP,O=cp.com.9jjkfo"; sequencenum:"62"; time:"1620217629"; version:"5"; __policy_id_tag:"product=VPN-1 & FireWall-1[db_tag={F6212FB3-54CE-6344-9164-B224119E2B92};mgmt=cp-m;date=1620031791;policy_name=CP-Cluster]"; action_reason:"Dropped by multiportal infrastructure"; dst:"81.2.69.144"; product:"VPN & FireWall"; proto:"6"; s_port:"52780"; service:"80"; src:"81.2.69.144"]`,
			Want: message{
				timestamp:  mustParseTime(time.RFC3339Nano, "2003-10-11T22:14:15.003Z", nil),
				priority:   165,
				facility:   20,
				severity:   5,
				version:    1,
				hostname:   "mymachine.example.com",
				process:    "evntslog",
				msgID:      "ID47",
				rawSDValue: `[action:"Drop"; flags:"278528"; ifdir:"inbound"; ifname:"bond1.3999"; loguid:"{0x60928f1d,0x8,0x40de101f,0xfcdbb197}"; origin:"127.0.0.1"; originsicname:"CN=CP,O=cp.com.9jjkfo"; sequencenum:"62"; time:"1620217629"; version:"5"; __policy_id_tag:"product=VPN-1 & FireWall-1[db_tag={F6212FB3-54CE-6344-9164-B224119E2B92};mgmt=cp-m;date=1620031791;policy_name=CP-Cluster]"; action_reason:"Dropped by multiportal infrastructure"; dst:"81.2.69.144"; product:"VPN & FireWall"; proto:"6"; s_port:"52780"; service:"80"; src:"81.2.69.144"]`,
			},
		},
		"non-compliant-sd-with-msg": {
			In: `<165>1 2003-10-11T22:14:15.003Z mymachine.example.com evntslog - ID47 [action:"Drop"; flags:"278528"; ifdir:"inbound"; ifname:"bond1.3999"; loguid:"{0x60928f1d,0x8,0x40de101f,0xfcdbb197}"; origin:"127.0.0.1"; originsicname:"CN=CP,O=cp.com.9jjkfo"; sequencenum:"62"; time:"1620217629"; version:"5"; __policy_id_tag:"product=VPN-1 & FireWall-1[db_tag={F6212FB3-54CE-6344-9164-B224119E2B92};mgmt=cp-m;date=1620031791;policy_name=CP-Cluster]"; action_reason:"Dropped by multiportal infrastructure"; dst:"81.2.69.144"; product:"VPN & FireWall"; proto:"6"; s_port:"52780"; service:"80"; src:"81.2.69.144"] This is a test message`,
			Want: message{
				timestamp:  mustParseTime(time.RFC3339Nano, "2003-10-11T22:14:15.003Z", nil),
				priority:   165,
				facility:   20,
				severity:   5,
				version:    1,
				hostname:   "mymachine.example.com",
				process:    "evntslog",
				msgID:      "ID47",
				rawSDValue: `[action:"Drop"; flags:"278528"; ifdir:"inbound"; ifname:"bond1.3999"; loguid:"{0x60928f1d,0x8,0x40de101f,0xfcdbb197}"; origin:"127.0.0.1"; originsicname:"CN=CP,O=cp.com.9jjkfo"; sequencenum:"62"; time:"1620217629"; version:"5"; __policy_id_tag:"product=VPN-1 & FireWall-1[db_tag={F6212FB3-54CE-6344-9164-B224119E2B92};mgmt=cp-m;date=1620031791;policy_name=CP-Cluster]"; action_reason:"Dropped by multiportal infrastructure"; dst:"81.2.69.144"; product:"VPN & FireWall"; proto:"6"; s_port:"52780"; service:"80"; src:"81.2.69.144"]`,
				msg:        "This is a test message",
			},
		},
		"err-invalid-version": {
			In: `<165>A 2003-10-11T22:14:15.003Z mymachine.example.com evntslog - ID47 [exampleSDID@32473 iut="3" eventSource="Application" eventID="1011"][examplePriority@32473 class="high"]`,
			Want: message{
				timestamp:  mustParseTime(time.RFC3339Nano, "2003-10-11T22:14:15.003Z", nil),
				priority:   165,
				facility:   20,
				severity:   5,
				hostname:   "mymachine.example.com",
				process:    "evntslog",
				msgID:      "ID47",
				rawSDValue: `[exampleSDID@32473 iut="3" eventSource="Application" eventID="1011"][examplePriority@32473 class="high"]`,
			},
			WantErr: `validation error at position 6: invalid version, expected an integer: strconv.Atoi: parsing "A": invalid syntax`,
		},
		"err-invalid-timestamp": {
			In: `<165>1 10-11-2003T22:14:15.003Z mymachine.example.com evntslog - ID47 [exampleSDID@32473 iut="3" eventSource="Application" eventID="1011"][examplePriority@32473 class="high"]`,
			Want: message{
				priority:   165,
				facility:   20,
				severity:   5,
				version:    1,
				hostname:   "mymachine.example.com",
				process:    "evntslog",
				msgID:      "ID47",
				rawSDValue: `[exampleSDID@32473 iut="3" eventSource="Application" eventID="1011"][examplePriority@32473 class="high"]`,
			},
			WantErr: `validation error at position 8: parsing time "10-11-2003T22:14:15.003Z" as "2006-01-02T15:04:05.999999999Z07:00": cannot parse "1-2003T22:14:15.003Z" as "2006"`,
		},
		"err-eof": {
			In: `<13>1 2003-08-24T05:14:15.000003-07:00 test-host su 1234 msg-`,
			Want: message{
				timestamp: mustParseTime(time.RFC3339Nano, "2003-08-24T05:14:15.000003-07:00", nil),
				priority:  13,
				facility:  1,
				severity:  5,
				hostname:  "test-host",
				process:   "su",
				pid:       "1234",
				version:   1,
			},
			WantErr: `parsing error at position 62: message is truncated (unexpected EOF)`,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, gotErr := parseRFC5424(tc.In)

			if tc.WantErr != "" {
				assert.ErrorContains(t, gotErr, tc.WantErr)
			} else {
				assert.NoError(t, gotErr)
			}

			assert.Equal(t, tc.Want, got)
		})
	}
}

func BenchmarkParseRFC5424(b *testing.B) {
	tests := map[string]struct {
		In string
	}{
		"example-1": {
			In: "<13>1 2003-08-24T05:14:15.000003-07:00 test-host su 1234 msg-5678 - This is a test message",
		},
		"example-2": {
			In: `<13>1 2003-08-24T05:14:15.000003-07:00 test-host su 1234 msg-5678 [sd-id-1 foo="bar"] This is a test message`,
		},
		"example-3": {
			In: `<13>1 - - - - - -`,
		},
		"example-4": {
			In: `<34>1 2003-10-11T22:14:15.003Z mymachine.example.com su - ID47 - ` + utf8BOM + `'su root' failed for user1 on /dev/pts/8`,
		},
		"example-5": {
			In: `<165>1 2003-10-11T22:14:15.003Z mymachine.example.com evntslog - ID47 [exampleSDID@32473 iut="3" eventSource="Application" eventID="1011"][examplePriority@32473 class="high"]`,
		},
	}

	for name, bc := range tests {
		bc := bc
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = parseRFC5424(bc.In)
			}
		})
	}
}

func TestIsRFC5424(t *testing.T) {
	tests := map[string]struct {
		In   string
		Want bool
	}{
		"rfc-5424": {
			In:   "<13>1 2003-08-24T05:14:15.000003-07:00 test-host su 1234 msg-5678 - This is a test message",
			Want: true,
		},
		"rfc-3164": {
			In:   "<13>Oct 11 22:14:15 test-host this is the message",
			Want: false,
		},
		"invalid-message": {
			In:   "not a valid message",
			Want: false,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := isRFC5424(tc.In)

			assert.Equal(t, tc.Want, got)
		})
	}
}

func BenchmarkIsRFC5424(b *testing.B) {
	tests := map[string]struct {
		In string
	}{
		"rfc-5424": {
			In: "<13>1 2003-08-24T05:14:15.000003-07:00 test-host su 1234 msg-5678 - This is a test message",
		},
		"rfc-3164": {
			In: "<13>Oct 11 22:14:15 test-host this is the message",
		},
		"invalid-message": {
			In: "not a valid message",
		},
	}

	for name, bc := range tests {
		bc := bc
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = isRFC5424(bc.In)
			}
		})
	}
}
