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

func TestParseRFC3164(t *testing.T) {
	tests := map[string]struct {
		in      string
		want    message
		wantErr string
	}{
		"ok": {
			in: "<13>Oct 11 22:14:15 test-host this is the message",
			want: message{
				timestamp: mustParseTime(time.Stamp, "Oct 11 22:14:15", time.Local),
				priority:  13,
				facility:  1,
				severity:  5,
				hostname:  "test-host",
				msg:       "this is the message",
			},
		},
		"ok-rfc3339": {
			in: "<13>2003-08-24T05:14:15.000003-07:00 test-host this is the message",
			want: message{
				timestamp: mustParseTime(time.RFC3339Nano, "2003-08-24T05:14:15.000003-07:00", nil),
				priority:  13,
				facility:  1,
				severity:  5,
				hostname:  "test-host",
				msg:       "this is the message",
			},
		},
		"ok-process": {
			in: "<13>Oct 11 22:14:15 test-host su: this is the message",
			want: message{
				timestamp: mustParseTime(time.Stamp, "Oct 11 22:14:15", time.Local),
				priority:  13,
				facility:  1,
				severity:  5,
				hostname:  "test-host",
				process:   "su",
				msg:       "this is the message",
			},
		},
		"ok-process-pid": {
			in: "<13>Oct 11 22:14:15 test-host su[1024]: this is the message",
			want: message{
				timestamp: mustParseTime(time.Stamp, "Oct 11 22:14:15", time.Local),
				priority:  13,
				facility:  1,
				severity:  5,
				hostname:  "test-host",
				process:   "su",
				pid:       "1024",
				msg:       "this is the message",
			},
		},
		"non-standard-date": {
			in: "<123>Sep 01 02:03:04 hostname message",
			want: message{
				timestamp: mustParseTime(time.Stamp, "Sep 1 02:03:04", time.Local),
				priority:  123,
				facility:  15,
				severity:  3,
				hostname:  "hostname",
				msg:       "message",
			},
		},
		"ok-procid-with-square-brackets-msg": {
			in: "<114>Apr 12 13:30:01 aaaaaa001.adm.domain aaaaaa001[25259]: my.some.domain 10.11.12.13 - USERNAME [12/Apr/2024:13:29:59.993 +0200] /skodas \"GET /skodas/group/pod-documentation/aaa HTTP/1.1\" 301 301 290bytes 1 10327",
			want: message{
				timestamp: mustParseTime(time.Stamp, "Apr 12 13:30:01", time.Local),
				priority:  114,
				facility:  14,
				severity:  2,
				hostname:  "aaaaaa001.adm.domain",
				process:   "aaaaaa001",
				pid:       "25259",
				msg:       "my.some.domain 10.11.12.13 - USERNAME [12/Apr/2024:13:29:59.993 +0200] /skodas \"GET /skodas/group/pod-documentation/aaa HTTP/1.1\" 301 301 290bytes 1 10327",
			},
		},
		"err-pri-not-a-number": {
			in: "<abc>Oct 11 22:14:15 test-host this is the message",
			want: message{
				timestamp: mustParseTime(time.Stamp, "Oct 11 22:14:15", time.Local),
				priority:  -1,
				hostname:  "test-host",
				msg:       "this is the message",
			},
			wantErr: `validation error at position 2: invalid priority: strconv.Atoi: parsing "abc": invalid syntax`,
		},
		"err-pri-out-of-range": {
			in: "<192>Oct 11 22:14:15 test-host this is the message",
			want: message{
				timestamp: mustParseTime(time.Stamp, "Oct 11 22:14:15", time.Local),
				priority:  -1,
				hostname:  "test-host",
				msg:       "this is the message",
			},
			wantErr: `validation error at position 2: priority value out of range (expected 0..191)`,
		},
		"err-pri-negative": {
			in: "<-1>Oct 11 22:14:15 test-host this is the message",
			want: message{
				timestamp: mustParseTime(time.Stamp, "Oct 11 22:14:15", time.Local),
				priority:  -1,
				hostname:  "test-host",
				msg:       "this is the message",
			},
			wantErr: `validation error at position 2: priority value out of range (expected 0..191)`,
		},
		"err-pri-missing-brackets": {
			in: "13 Oct 11 22:14:15 test-host this is the message",
			want: message{
				priority: -1,
				hostname: "Oct",
				msg:      "11 22:14:15 test-host this is the message",
			},
			wantErr: `validation error at position 1: parsing time "13" as "2006-01-02T15:04:05.999999999Z07:00": cannot parse "13" as "2006"`,
		},
		"err-ts-invalid-missing": {
			in: "<13> test-host this is the message",
			want: message{
				priority: 13,
				facility: 1,
				severity: 5,
			},
			wantErr: `parsing error at position 5: unexpected EOF`,
		},
		"err-ts-invalid-bsd": {
			in: "<13>Foo 11 22:14:15 test-host this is the message",
			want: message{
				priority: 13,
				facility: 1,
				severity: 5,
				hostname: "test-host",
				msg:      "this is the message",
			},
			wantErr: `validation error at position 5: parsing time "Foo 11 22:14:15" as "Jan _2 15:04:05": cannot parse "Foo 11 22:14:15" as "Jan"`,
		},
		"err-ts-invalid-rfc3339": {
			in: "<13>24-08-2003T05:14:15-07:00 test-host this is the message",
			want: message{
				priority: 13,
				facility: 1,
				severity: 5,
				hostname: "test-host",
				msg:      "this is the message",
			},
			wantErr: `validation error at position 5: parsing time "24-08-2003T05:14:15-07:00" as "2006-01-02T15:04:05.999999999Z07:00": cannot parse "24-08-2003T05:14:15-07:00" as "2006"`,
		},
		"err-eof": {
			in: "<13>Oct 11 22:14:15 test-",
			want: message{
				timestamp: mustParseTime(time.Stamp, "Oct 11 22:14:15", time.Local),
				priority:  13,
				facility:  1,
				severity:  5,
			},
			wantErr: `parsing error at position 26: unexpected EOF`,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, gotErr := parseRFC3164(tc.in, time.Local)

			if tc.wantErr != "" {
				assert.ErrorContains(t, gotErr, tc.wantErr)
			} else {
				assert.NoError(t, gotErr)
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

func BenchmarkParseRFC3164(b *testing.B) {
	tests := map[string]struct {
		in string
	}{
		"ok": {
			in: "<13>Oct 11 22:14:15 test-host this is the message",
		},
		"ok-rfc3339": {
			in: "<13>2003-08-24T05:14:15.000003-07:00 test-host this is the message",
		},
		"ok-process": {
			in: "<13>Oct 11 22:14:15 test-host su: this is the message",
		},
		"ok-process-pid": {
			in: "<13>Oct 11 22:14:15 test-host su[1024]: this is the message",
		},
		"non-standard-date": {
			in: "<123>Sep 01 02:03:04 hostname message",
		},
		"err-pri-not-a-number": {
			in: "<abc>Oct 11 22:14:15 test-host this is the message",
		},
		"err-pri-out-of-range": {
			in: "<192>Oct 11 22:14:15 test-host this is the message",
		},
		"err-pri-negative": {
			in: "<-1>Oct 11 22:14:15 test-host this is the message",
		},
		"err-pri-missing-brackets": {
			in: "13 Oct 11 22:14:15 test-host this is the message",
		},
		"err-ts-invalid-missing": {
			in: "<13> test-host this is the message",
		},
		"err-ts-invalid-bsd": {
			in: "<13>Foo 11 22:14:15 test-host this is the message",
		},
		"err-ts-invalid-rfc3339": {
			in: "<13>2003-08-24 05:14:15-07:00 test-host this is the message",
		},
	}

	for name, bc := range tests {
		bc := bc
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = parseRFC3164(bc.in, time.Local)
			}
		})
	}
}
